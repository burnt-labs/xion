package integration_tests

import (
	"fmt"
	"testing"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
)

func TestXionAbstractAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.Msg)(nil),
		&xiontypes.MsgSetPlatformPercentage{},
		&xiontypes.MsgSend{},
		&wasmtypes.MsgInstantiateContract{},
		&wasmtypes.MsgStoreCode{},
		&aatypes.MsgUpdateParams{},
		&aatypes.MsgRegisterAccount{},
	)

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	currentHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// step 1: send a xion message with default (0%) platform fee
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)

	/*
		receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
		require.NoError(t, err)
		recipientKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
		require.NoError(t, err)
	*/

	currentHeight, _ = xion.Height(ctx)
	account, err := ExecBin(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"keys", "show",
		xionUser.KeyName(),
		"--keyring-backend", keyring.BackendTest,
		"-p",
	)
	require.NoError(t, err)
	fmt.Println("Post Query")
	fmt.Println("=======================================================")
	fmt.Println(account)
	fmt.Println("=======================================================")

	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), "./testdata/contracts/account_updatable-aarch64.wasm")
	require.NoError(t, err)

	/* add register AA using Public key */
	registeredTxHash, err := ExecTx(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"abstract-account", "register",
		codeID,
		fmt.Sprintf(`{"pubkey": "%s"}`, account["key"]),
		"--funds", fmt.Sprintf("%d%s", 100000, xion.Config().Denom),
		"--salt", "foo",
		"--chain-id", xion.Config().ChainID,
	)

	txDetails, err := ExecQuery(t, ctx, xion.FullNodes[0], "tx", registeredTxHash)
	require.NoError(t, err)
	fmt.Println("Post Register?")
	fmt.Println("=======================================================")
	fmt.Println(r)
	fmt.Println("=======================================================")

	//_, err = xion.InstantiateContract(ctx, xionUser.FormattedAddress(), codeID, fmt.Sprintf(`{"new_pubkey:%s"}`, ""), true)

	/*
		require.NoError(t, err)
		balance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, uint64(100), uint64(balance))
	*/
}
