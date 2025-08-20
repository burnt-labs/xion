package cli_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	wasm "github.com/CosmWasm/wasmd/x/wasm"
	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"

	feegrantmod "cosmossdk.io/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authzmod "github.com/cosmos/cosmos-sdk/x/authz/module"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	gov "github.com/cosmos/cosmos-sdk/x/gov"
	staking "github.com/cosmos/cosmos-sdk/x/staking"

	"github.com/burnt-labs/xion/x/xion/client/cli"
)

// makeClientCtx builds a lightweight client context with a single ephemeral key.
func makeClientCtx(t *testing.T) (client.Context, string, sdk.AccAddress) {
	enc := testutilmod.MakeTestEncodingConfig(bank.AppModuleBasic{}, feegrantmod.AppModuleBasic{}, authzmod.AppModuleBasic{}, staking.AppModuleBasic{}, gov.AppModuleBasic{}, wasm.AppModuleBasic{})
	kr := keyring.NewInMemory(enc.Codec)
	name := "k1"
	_, mnemonic, err := kr.NewMnemonic(name, keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	require.NoError(t, err)
	_ = mnemonic
	rec, err := kr.Key(name)
	require.NoError(t, err)
	addr, err := rec.GetAddress()
	require.NoError(t, err)
	ctx := client.Context{}.
		WithChainID("test-chain").
		WithKeyring(kr).
		WithFromName(name).
		WithFromAddress(addr).
		WithTxConfig(enc.TxConfig).
		WithCodec(enc.Codec).
		WithClient(clitestutil.MockCometRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(client.MockAccountRetriever{})
	return ctx, name, addr
}

func TestRegisterCmd_ErrorPaths(t *testing.T) {
	ctx, fromName, _ := makeClientCtx(t)
	cmd := cli.NewRegisterCmd()
	cmd.SetOut(os.Stdout)
	// invalid code id triggers parse error
	args := []string{"notuint", fromName}
	cmd.SetArgs(append(args,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		"--salt=abc",
		"--authenticator=Secp256K1",
		"--authenticator-id=1",
		"--funds=5uxion",
		"--generate-only",
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
	))
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContextHandler(ctx, cmd))
	err := cmd.Execute()
	require.Error(t, err)

	// valid code id but bad funds format triggers amount parse error path
	cmd = cli.NewRegisterCmd()
	cmd.SetOut(os.Stdout)
	args = []string{"1", fromName}
	cmd.SetArgs(append(args,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		"--salt=abc",
		"--authenticator=Secp256K1",
		"--authenticator-id=2",
		"--funds=badcoins",
		"--generate-only",
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
	))
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContextHandler(ctx, cmd))
	err = cmd.Execute()
	require.Error(t, err)
}

func TestAddAuthenticatorCmd_GenerateOnly(t *testing.T) {
	ctx, fromName, addr := makeClientCtx(t)
	cmd := cli.NewAddAuthenticatorCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetArgs([]string{
		addr.String(),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		"--authenticator-id=1",
		"--generate-only",
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	})
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContextHandler(ctx, cmd))
	_ = cmd.Execute()
}

func TestEmitArbitraryDataCmd_GenerateOnly(t *testing.T) {
	ctx, fromName, addr := makeClientCtx(t)
	cmd := cli.NewEmitArbitraryDataCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetArgs([]string{
		"payload", addr.String(),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		"--authenticator-id=1",
		"--generate-only",
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	})
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContextHandler(ctx, cmd))
	_ = cmd.Execute()
}

func TestUpdateParamsCmd_Scenarios(t *testing.T) {
	ctx, fromName, addr := makeClientCtx(t)
	// success-ish path (generate-only)
	cmd := cli.NewUpdateParamsCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetArgs([]string{
		addr.String(),
		"https://example.com/display",
		"https://example.com/redirect",
		"https://example.com/icon.png",
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		"--generate-only",
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	})
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContextHandler(ctx, cmd))
	_ = cmd.Execute()

	// invalid display URL triggers error
	cmd = cli.NewUpdateParamsCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetArgs([]string{
		addr.String(),
		":://bad", // invalid URL
		"https://example.com/redirect",
		"https://example.com/icon.png",
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		"--generate-only",
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	})
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContextHandler(ctx, cmd))
	err := cmd.Execute()
	require.Error(t, err)
}

func TestSignCmd_EarlyError(t *testing.T) {
	ctx, fromName, _ := makeClientCtx(t)
	cmd := cli.NewSignCmd()
	cmd.SetOut(os.Stdout)
	// bad signer bech32 so we stop before broadcast
	cmd.SetArgs([]string{
		fromName,
		"badbech32",
		"nonexistent.json",
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		"--authenticator-id=1",
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	})
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContextHandler(ctx, cmd))
	err := cmd.Execute()
	require.Error(t, err)
}
