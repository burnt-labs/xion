package cli_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/burnt-labs/barretenberg-go/barretenberg"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	"cosmossdk.io/log/v2"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/zk/client/cli"
	"github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
)

// groth16Proof and groth16PublicInputs verify against the default genesis
// verification key (types.CreateDefaultVKeyBytes).
const (
	groth16Proof        = `{"pi_a":["3520409830771234065994556634681427417950500323669026192927037346754953128831","3498073333995773007990629866509538975797753241506847777821035377435020235118","1"],"pi_b":[["5449376970517492854877490913806025222720139652422610862840169172156024987414","19970795329053509942253184076261379588636216808533565932150420541118191182415"],["3329994826233724633278876134031495025607573045416243454711110428426399497078","12635663937591831780654697557378573948535834769571974614171466954698620359415"],["1","0"]],"pi_c":["7013834827668081216347999320368574690710153592707686848856775126651375200226","10870223918887009180919296136906814662608691011484426082412176806715867878616","1"],"protocol":"groth16","curve":"bn128"}`
	groth16PublicInputs = "2018721414038404820327,0,0,0,0,0,0,0,0,6632353713085157925504008443078919716322386156160602218536961028046468237192,21084905302941062264575804210871487148258363738073263632230120817255351393954,0,191581113848055322477272311147821680130451026496941019613909483584263833445,149108628584424258332964971884436592255105616775526759101383287099246929273,20356082004311139738363494460884070443445370694676839,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,9079378704521501721378444251561135763203091338587747860525949554473799137061,1,145464208130933216679374873468710647147,0,0,0,0,0,0,0,0,180980592328871182281563474567090989367752380861661653173671556731952063826,189366407839159640650411313259066674300878650730387363415856879007716700777,112965544445135736799656303,0,0,0,0,0,0"
)

// keeperFixture reads a binary fixture from the keeper package's testdata.
func keeperFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "keeper", "testdata"}, parts...)...)
	bz, err := os.ReadFile(path)
	require.NoError(t, err)
	return bz
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

func requireRealBarretenberg(t *testing.T) {
	t.Helper()
	if strings.HasPrefix(barretenberg.Version(), "stub") {
		t.Skip("stub barretenberg library cannot validate UltraHonk keys")
	}
}

// queryNode answers ABCI queries in-process by routing them through a
// GRPCQueryRouter bound to a zk keeper, so the query commands run end to end
// without a node.
type queryNode struct {
	client.CometRPC

	ctx    sdk.Context
	router *baseapp.GRPCQueryRouter
}

func (n queryNode) ABCIQueryWithOptions(_ context.Context, path string, data cmtbytes.HexBytes, _ rpcclient.ABCIQueryOptions) (*coretypes.ResultABCIQuery, error) {
	handler := n.router.Route(path)
	if handler == nil {
		return nil, fmt.Errorf("unknown query path %s", path)
	}
	res, err := handler(n.ctx, &abci.RequestQuery{Path: path, Data: data})
	if err != nil {
		return nil, err
	}
	return &coretypes.ResultABCIQuery{Response: *res}, nil
}

type queryHarness struct {
	ctx       sdk.Context
	keeper    keeper.Keeper
	clientCtx client.Context
	authority string
}

func newQueryHarness(t *testing.T) *queryHarness {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	k := keeper.NewKeeper(encCfg.Codec, runtime.NewKVStoreService(key), log.NewNopLogger(), authority)
	require.NoError(t, k.SetParams(testCtx.Ctx, types.DefaultParams()))
	// Skip id 0 so the first key is addressable through --vkey-id, as on a
	// chain whose genesis seeds id 1.
	_, err := k.NextVKeyID.Next(testCtx.Ctx)
	require.NoError(t, err)

	router := baseapp.NewGRPCQueryRouter()
	router.SetInterfaceRegistry(encCfg.InterfaceRegistry)
	types.RegisterQueryServer(router, keeper.NewQuerier(k))

	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithClient(queryNode{ctx: testCtx.Ctx, router: router}).
		WithOutputFormat(flags.OutputFormatJSON)

	return &queryHarness{ctx: testCtx.Ctx, keeper: k, clientCtx: clientCtx, authority: authority}
}

func (h *queryHarness) addVKey(t *testing.T, name string, keyBytes []byte, proofSystem types.ProofSystem) uint64 {
	t.Helper()
	id, err := h.keeper.AddVKey(h.ctx, h.authority, name, keyBytes, name+" key", proofSystem)
	require.NoError(t, err)
	return id
}

func (h *queryHarness) exec(cmd *cobra.Command, args ...string) (string, error) {
	out, err := clitestutil.ExecTestCLICmd(h.clientCtx, cmd, args)
	return out.String(), err
}

func TestQueryCommandsAgainstKeeper(t *testing.T) {
	h := newQueryHarness(t)
	id := h.addVKey(t, "email_auth", types.CreateDefaultVKeyBytes(), types.ProofSystem_PROOF_SYSTEM_GROTH16)
	proofFile := writeTempFile(t, "proof.json", []byte(groth16Proof))

	t.Run("vkey by id", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryVKey(), fmt.Sprint(id))
		require.NoError(t, err)
		require.Contains(t, out, `"email_auth"`)
	})

	t.Run("vkey by id not found", func(t *testing.T) {
		_, err := h.exec(cli.GetCmdQueryVKey(), fmt.Sprint(id+100))
		require.ErrorContains(t, err, "not found")
	})

	t.Run("vkey by name", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryVKeyByName(), "email_auth")
		require.NoError(t, err)
		require.Contains(t, out, fmt.Sprintf(`"id":"%d"`, id))
	})

	t.Run("vkeys", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryVKeys(), "--"+flags.FlagLimit, "10")
		require.NoError(t, err)
		require.Contains(t, out, `"email_auth"`)
	})

	t.Run("has-vkey", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryHasVKey(), "email_auth")
		require.NoError(t, err)
		require.Contains(t, out, `"exists":true`)
	})

	t.Run("params", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryParams())
		require.NoError(t, err)
		require.Contains(t, out, `"max_vkey_size_bytes"`)
	})

	t.Run("verify-proof by name", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryVerifyProof(), proofFile, "--vkey-name", "email_auth", "--public-inputs", groth16PublicInputs)
		require.NoError(t, err)
		require.Contains(t, out, `"verified":true`)
	})

	t.Run("verify-proof by id", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryVerifyProof(), proofFile, "--vkey-id", fmt.Sprint(id), "--public-inputs", groth16PublicInputs)
		require.NoError(t, err)
		require.Contains(t, out, `"verified":true`)
	})

	t.Run("verify-proof with unknown key", func(t *testing.T) {
		_, err := h.exec(cli.GetCmdQueryVerifyProof(), proofFile, "--vkey-name", "missing", "--public-inputs", groth16PublicInputs)
		require.ErrorContains(t, err, "missing")
	})
}

func TestQueryVerifyUltraHonkAgainstKeeper(t *testing.T) {
	requireRealBarretenberg(t)

	h := newQueryHarness(t)
	h.addVKey(t, "honk", keeperFixture(t, "barretenberg", "vk"), types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK)
	h.addVKey(t, "circom", types.CreateDefaultVKeyBytes(), types.ProofSystem_PROOF_SYSTEM_GROTH16)
	proofFile := writeTempFile(t, "proof.bin", keeperFixture(t, "barretenberg", "proof"))
	inputsFile := writeTempFile(t, "inputs.bin", keeperFixture(t, "barretenberg", "public_inputs"))

	t.Run("verifies against an ultrahonk key", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryVerifyUltraHonk(), proofFile, "--vkey-name", "honk", "--public-inputs-file", inputsFile)
		require.NoError(t, err)
		require.Contains(t, out, `"verified":true`)
	})

	t.Run("rejects a groth16 key", func(t *testing.T) {
		_, err := h.exec(cli.GetCmdQueryVerifyUltraHonk(), proofFile, "--vkey-name", "circom", "--public-inputs-file", inputsFile)
		require.ErrorContains(t, err, "not an UltraHonk key")
	})
}

func TestQueryVerifyGnarkAgainstKeeper(t *testing.T) {
	h := newQueryHarness(t)
	h.addVKey(t, "gnark", keeperFixture(t, "gnark", "vkey.bin"), types.ProofSystem_PROOF_SYSTEM_GROTH16_GNARK)
	h.addVKey(t, "circom", types.CreateDefaultVKeyBytes(), types.ProofSystem_PROOF_SYSTEM_GROTH16)
	proofFile := writeTempFile(t, "proof.bin", keeperFixture(t, "gnark", "proof.bin"))
	inputsFile := writeTempFile(t, "inputs.bin", keeperFixture(t, "gnark", "public_input.bin"))

	t.Run("verifies against a gnark key", func(t *testing.T) {
		out, err := h.exec(cli.GetCmdQueryVerifyGnark(), proofFile, "--vkey-name", "gnark", "--public-inputs-file", inputsFile)
		require.NoError(t, err)
		require.Contains(t, out, `"verified":true`)
	})

	t.Run("rejects a groth16 key", func(t *testing.T) {
		_, err := h.exec(cli.GetCmdQueryVerifyGnark(), proofFile, "--vkey-name", "circom", "--public-inputs-file", inputsFile)
		require.ErrorContains(t, err, "not a gnark Groth16 key")
	})
}

func TestQueryCommandsRejectInvalidNode(t *testing.T) {
	proofFile := writeTempFile(t, "proof", []byte("{}"))
	inputsFile := writeTempFile(t, "inputs", []byte{})

	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"vkey", cli.GetCmdQueryVKey, []string{"1"}},
		{"vkey-by-name", cli.GetCmdQueryVKeyByName, []string{"name"}},
		{"vkeys", cli.GetCmdQueryVKeys, nil},
		{"has-vkey", cli.GetCmdQueryHasVKey, []string{"name"}},
		{"verify-proof", cli.GetCmdQueryVerifyProof, []string{proofFile, "--vkey-name", "name", "--public-inputs", "1"}},
		{"verify-ultrahonk", cli.GetCmdQueryVerifyUltraHonk, []string{proofFile, "--vkey-name", "name", "--public-inputs-file", inputsFile}},
		{"verify-gnark", cli.GetCmdQueryVerifyGnark, []string{proofFile, "--vkey-name", "name", "--public-inputs-file", inputsFile}},
		{"params", cli.GetCmdQueryParams, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmd()
			cmd.SetArgs(append(tc.args, "--"+flags.FlagNode, "://"))
			err := cmd.Execute()
			require.ErrorContains(t, err, "missing protocol scheme")
		})
	}
}

func TestQueryCommandArgumentErrors(t *testing.T) {
	t.Run("vkey rejects a non numeric id", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKey()
		cmd.SetArgs([]string{"not-a-number"})
		err := cmd.Execute()
		require.ErrorContains(t, err, "invalid vkey ID")
	})

	t.Run("vkeys rejects page combined with offset", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeys()
		cmd.SetArgs([]string{"--" + flags.FlagPage, "2", "--" + flags.FlagOffset, "1"})
		err := cmd.Execute()
		require.ErrorContains(t, err, "page and offset cannot be used together")
	})
}

func TestQueryCommandsSurfaceNodeErrors(t *testing.T) {
	h := newQueryHarness(t)

	t.Run("has-vkey with an empty name", func(t *testing.T) {
		_, err := h.exec(cli.GetCmdQueryHasVKey(), "")
		require.ErrorContains(t, err, "name cannot be empty")
	})

	t.Run("vkeys when the node cannot serve the query", func(t *testing.T) {
		unrouted := h.clientCtx.WithClient(queryNode{ctx: h.ctx, router: baseapp.NewGRPCQueryRouter()})
		_, err := clitestutil.ExecTestCLICmd(unrouted, cli.GetCmdQueryVKeys(), nil)
		require.ErrorContains(t, err, "unknown query path")
	})
}
