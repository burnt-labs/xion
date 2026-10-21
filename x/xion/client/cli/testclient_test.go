package cli_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/testutil"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authz "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant/module"

	aatypes "github.com/burnt-labs/xion/x/abstractaccount/types"
)

var _ client.CometRPC = routedCometRPC{}

// routedCometRPC answers ABCI queries by gRPC method path and lets a test
// choose how broadcasting behaves, so tx commands can run end to end without
// a node. Paths without a configured response fail the query.
type routedCometRPC struct {
	rpcclientmock.Client

	responses    map[string]abci.ResponseQuery
	broadcastErr error
}

func (m routedCometRPC) ABCIQueryWithOptions(
	_ context.Context,
	path string,
	_ cmtbytes.HexBytes,
	_ rpcclient.ABCIQueryOptions,
) (*coretypes.ResultABCIQuery, error) {
	if resp, ok := m.responses[path]; ok {
		return &coretypes.ResultABCIQuery{Response: resp}, nil
	}
	return &coretypes.ResultABCIQuery{Response: abci.ResponseQuery{
		Code:      1,
		Codespace: "test",
		Log:       "no response configured for " + path,
	}}, nil
}

func (m routedCometRPC) BroadcastTxSync(context.Context, cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	if m.broadcastErr != nil {
		return nil, m.broadcastErr
	}
	return &coretypes.ResultBroadcastTx{}, nil
}

// queryResponse encodes msg the way a node would return it from an ABCI query.
func queryResponse(t *testing.T, msg proto.Message) abci.ResponseQuery {
	t.Helper()
	bz, err := proto.Marshal(msg)
	require.NoError(t, err)
	return abci.ResponseQuery{Value: bz}
}

// cliTestEnv holds a keyring with one funded-looking account and an encoding
// config that knows every message the xion tx commands produce.
type cliTestEnv struct {
	encCfg  testutilmod.TestEncodingConfig
	kr      keyring.Keyring
	account testutil.TestAccount
	out     *bytes.Buffer
}

func newCLITestEnv(t *testing.T) cliTestEnv {
	t.Helper()
	encCfg := testutilmod.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		feegrant.AppModuleBasic{},
		authz.AppModuleBasic{},
		wasm.AppModuleBasic{},
	)
	aatypes.RegisterInterfaces(encCfg.InterfaceRegistry)

	kr := keyring.NewInMemory(encCfg.Codec)
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)

	return cliTestEnv{encCfg: encCfg, kr: kr, account: accounts[0], out: &bytes.Buffer{}}
}

func (e cliTestEnv) clientContext(rpc client.CometRPC) client.Context {
	return client.Context{}.
		WithKeyring(e.kr).
		WithTxConfig(e.encCfg.TxConfig).
		WithCodec(e.encCfg.Codec).
		WithInterfaceRegistry(e.encCfg.InterfaceRegistry).
		WithClient(rpc).
		WithAccountRetriever(client.MockAccountRetriever{}).
		WithChainID("test-chain").
		WithOutput(e.out)
}

// runCmd executes cmd the way the root command would: cobra parses args and
// flags, then the handler reads the client context from the command context.
func runCmd(t *testing.T, cmd *cobra.Command, clientCtx client.Context, args ...string) error {
	t.Helper()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetContext(svrcmd.CreateExecuteContext(context.Background()))
	cmd.SetArgs(args)
	require.NoError(t, client.SetCmdClientContextHandler(clientCtx, cmd))
	return cmd.Execute()
}

// capturePanic runs fn and returns the error it panicked with.
func capturePanic(t *testing.T, fn func()) error {
	t.Helper()
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		fn()
	}()
	err, ok := recovered.(error)
	require.True(t, ok, "expected a panic carrying an error, got %v", recovered)
	return err
}
