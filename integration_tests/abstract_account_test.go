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

	currentHeight, _ = xion.Height(ctx)
	account, err := ExecBin(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"keys", "show",
		xionUser.KeyName(),
		"--keyring-backend", keyring.BackendTest,
		"-p",
	)
	require.NoError(t, err)

	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), "./testdata/contracts/account_updatable-aarch64.wasm")
	require.NoError(t, err)

	depositedFunds := fmt.Sprintf("%d%s", 100000, xion.Config().Denom)

	/* add register AA using Public key */
	registeredTxHash, err := ExecTx(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"abstract-account", "register",
		codeID,
		fmt.Sprintf(`{"pubkey": "%s"}`, account["key"]),
		"--funds", depositedFunds,
		"--salt", "foo",
		"--chain-id", xion.Config().ChainID,
	)

	txDetails, err := ExecQuery(t, ctx, xion.FullNodes[0], "tx", registeredTxHash)
	require.NoError(t, err)
	aaContractAddr := GetAAContractAddress(t, txDetails)
	fmt.Println(aaContractAddr)
	fmt.Println("=======================================================")

	contractBalance, err := xion.GetBalance(ctx, aaContractAddr, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100000), uint64(contractBalance))

	/*
			NOTE: Ideally we would use this metod, however the QueryContract formats the string making it harder to predict.
		var ContractResponse interface{}
			require.NoError(t, xion.QueryContract(ctx, aaContractAddr, fmt.Sprintf(`{"pubkey":{}}`), ContractResponse))
	*/

	contractState, err := ExecQuery(t, ctx, xion.FullNodes[0], "wasm", "contract-state", "smart", aaContractAddr, fmt.Sprintf(`{"pubkey":{}}`))
	require.NoError(t, err)

	pubkey, ok := contractState["data"].(string)
	require.True(t, ok)
	require.Equal(t, account["key"], pubkey)

	// TODO:
	// - sign and rotate key

}

func GetAAContractAddress(t *testing.T, txDetails map[string]interface{}) string {
	logs, ok := txDetails["logs"].([]interface{})
	require.True(t, ok)

	log, ok := logs[0].(map[string]interface{})
	require.True(t, ok)

	events, ok := log["events"].([]interface{})
	require.True(t, ok)

	event, ok := events[4].(map[string]interface{})
	require.True(t, ok)

	attributes, ok := event["attributes"].([]interface{})
	require.True(t, ok)

	attribute, ok := attributes[0].(map[string]interface{})
	require.True(t, ok)

	addr, ok := attribute["value"].(string)
	require.True(t, ok)

	return addr
}
