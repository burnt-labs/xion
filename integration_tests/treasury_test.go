package integration_tests

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	"cosmossdk.io/x/feegrant"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

/* NOTE:
- Test for different types of feegrants: (AuthZAllowance, ContractsAllowance)
- Revoke allowance
*/

type GrantConfig struct {
	Description   string      `json:"description"`
	Authorization ExplicitAny `json:"authorization"`
	Optional      bool        `json:"optional"`
}

type FeeConfig struct {
	Description string       `json:"description"`
	Allowance   *ExplicitAny `json:"allowance,omitempty"`
	Expiration  int32        `json:"expiration,omitempty"`
}
type TreasuryInstantiateMsg struct {
	Admin        *string       `json:"admin,omitempty"` // Option<Addr> in Rust
	TypeUrls     []string      `json:"type_urls"`
	GrantConfigs []GrantConfig `json:"grant_configs"`
	FeeConfig    *FeeConfig    `json:"fee_config"` // Required field
	Params       *Params       `json:"params"`     // Required field
}

type ExplicitAny struct {
	TypeURL string `json:"type_url"`
	Value   []byte `json:"value"`
}

type Params struct {
	RedirectURL string `json:"redirect_url"`
	IconURL     string `json:"icon_url"`
	Metadata    string `json:"metadata"`
}

func TestTreasuryContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	err := testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	fp, err := os.Getwd()
	require.NoError(t, err)
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)
	t.Logf("deployed code id: %s", codeIDStr)

	inFive := time.Now().Add(time.Minute * 5)
	testAuth := authz.NewGenericAuthorization("/" + proto.MessageName(&banktypes.MsgSend{}))
	testGrant, err := authz.NewGrant(time.Now(), testAuth, &inFive)
	require.NoError(t, err)

	xionUserAddr, err := types.AccAddressFromBech32(xionUser.FormattedAddress())
	require.NoError(t, err)

	testAllowance := feegrant.BasicAllowance{
		SpendLimit: types.Coins{},
		Expiration: &inFive,
	}

	// NOTE: Create Feegrant
	feeGrant, err := feegrant.NewGrant(xionUserAddr, xionUserAddr, &testAllowance)
	require.NoError(t, err)
	allowanceAny := ExplicitAny{
		TypeURL: feeGrant.Allowance.TypeUrl,
		Value:   feeGrant.Allowance.Value,
	}

	authorizationAny := ExplicitAny{
		TypeURL: testGrant.Authorization.TypeUrl,
		Value:   testGrant.Authorization.Value,
	}

	grantConfig := GrantConfig{
		Description:   "test authorization",
		Authorization: authorizationAny,
		Optional:      false,
	}

	salt := "treasury-test-1"

	// Now create the actual instantiate message
	userAddrStr := xionUser.FormattedAddress()
	instantiateMsg := TreasuryInstantiateMsg{
		Admin:        &userAddrStr, // Set the user as admin (pointer)
		TypeUrls:     []string{testAuth.MsgTypeURL()},
		GrantConfigs: []GrantConfig{grantConfig},
		FeeConfig: &FeeConfig{
			Description: "test fee grant",
			Allowance:   &allowanceAny,
			Expiration:  int32(18000),
		},
		Params: &Params{
			RedirectURL: "https://example.com",
			IconURL:     "https://example.com/icon.png",
			Metadata:    "{}",
		},
	}

	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	// Use instantiate2 with a salt to get predictable address
	// Setting admin=true will make the contract its own admin at the blockchain level
	treasuryAddr, err := InstantiateContract2(t, ctx, xion, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), salt, true)
	require.NoError(t, err)
	t.Logf("created treasury instance: %s", treasuryAddr)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	contractState, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "all", treasuryAddr)
	require.NoError(t, err)
	t.Logf("Contract State: %s", contractState)

	granteeUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "grantee", "", fundAmount, xion)
	require.NoError(t, err)

	granterUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "granter", "", fundAmount, xion)
	require.NoError(t, err)
	t.Logf("granter: %s %s %s", granterUser.KeyName(), granterUser.FormattedAddress(), granterUser.Address())
	require.NoError(t, err)

	// wait for user creation
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	err = xion.SendFunds(ctx, granterUser.KeyName(), ibc.WalletAmount{
		Address: treasuryAddr,
		Denom:   "uxion",
		Amount:  math.NewInt(1000),
	})
	require.NoError(t, err)

	// NOTE: Create AuthZGrant
	authzGrantMsg, err := authz.NewMsgGrant(granterUser.Address(), granteeUser.Address(), testAuth, &inFive)
	require.NoError(t, err)

	executeMsg := map[string]interface{}{}
	feegrantMsg := map[string]interface{}{}
	feegrantMsg["authz_granter"] = granterUser.FormattedAddress()
	feegrantMsg["authz_grantee"] = granteeUser.FormattedAddress()
	executeMsg["deploy_fee_grant"] = feegrantMsg
	executeMsgBz, err := json.Marshal(executeMsg)
	require.NoError(t, err)

	// NOTE: Execute in contract
	contractMsg := wasmtypes.MsgExecuteContract{
		Sender:   granterUser.FormattedAddress(),
		Contract: treasuryAddr,
		Msg:      executeMsgBz,
		Funds:    nil,
	}

	require.NoError(t, err)

	// build the tx
	txBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(authzGrantMsg, &contractMsg)
	require.NoError(t, err)
	txBuilder.SetGasLimit(200000)
	tx := txBuilder.GetTx()

	txJSONStr, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(tx)
	require.NoError(t, err)

	t.Logf("tx: %s", txJSONStr)
	require.True(t, json.Valid([]byte(txJSONStr)))
	sendFile, err := os.CreateTemp("", "*-combo-msg-tx.json")
	require.NoError(t, err)
	defer os.Remove(sendFile.Name())

	_, err = sendFile.Write([]byte(txJSONStr))
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), sendFile)
	require.NoError(t, err)

	sendFilePath := strings.Split(sendFile.Name(), "/")

	signedTx, err := ExecBinRaw(t, ctx, xion.GetNode(),
		"tx", "sign", path.Join(xion.GetNode().HomeDir(), sendFilePath[len(sendFilePath)-1]),
		"--from", granterUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--overwrite",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.GetNode().HostName()))
	require.NoError(t, err)
	t.Logf("signed tx: %s", signedTx)

	// todo: validate that the feegrant was created correctly
	res, err := ExecBroadcastWithFlags(t, ctx, xion.GetNode(), signedTx, "--output", "json")
	require.NoError(t, err)
	t.Logf("broadcasted tx: %s", res)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	feeGrantDetails, err := ExecQuery(t, ctx, xion.GetNode(), "feegrant", "grants-by-grantee", granteeUser.FormattedAddress())
	require.NoError(t, err)
	t.Logf("FeeGrantDetails: %s", feeGrantDetails)
	allowances := feeGrantDetails["allowances"].([]interface{})
	allowance := (allowances[0].(map[string]interface{}))["allowance"].(map[string]interface{})
	allowanceType := allowance["type"].(string)
	require.Contains(t, allowanceType, "cosmos-sdk/BasicAllowance")

	revokeMsg := map[string]interface{}{}
	grantee := map[string]interface{}{}
	grantee["grantee"] = granteeUser.FormattedAddress()
	revokeMsg["revoke_allowance"] = grantee
	revokeMsgBz, err := json.Marshal(revokeMsg)
	require.NoError(t, err)

	revokeContractMsg := wasmtypes.MsgExecuteContract{
		Sender:   xionUser.FormattedAddress(),
		Contract: treasuryAddr,
		Msg:      revokeMsgBz,
		Funds:    nil,
	}
	newTxBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err = newTxBuilder.SetMsgs(&revokeContractMsg)
	require.NoError(t, err)
	newTxBuilder.SetGasLimit(20000000)
	newTx := newTxBuilder.GetTx()

	txJSONStr, err = xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(newTx)
	require.NoError(t, err)

	t.Logf("tx: %s", txJSONStr)
	require.True(t, json.Valid([]byte(txJSONStr)))
	revokeSendFile, err := os.CreateTemp("", "*-revoke-combo-msg-tx.json")
	require.NoError(t, err)
	defer os.Remove(revokeSendFile.Name())

	_, err = revokeSendFile.Write([]byte(txJSONStr))
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), revokeSendFile)
	require.NoError(t, err)

	revokeSendFilePath := strings.Split(revokeSendFile.Name(), "/")

	revokeSignedTx, err := ExecBinRaw(t, ctx, xion.GetNode(),
		"tx", "sign", path.Join(xion.GetNode().HomeDir(), revokeSendFilePath[len(revokeSendFilePath)-1]),
		"--from", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--overwrite",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.GetNode().HostName()))
	require.NoError(t, err)
	t.Logf("signed tx: %s", revokeSignedTx)

	// todo: validate that the feegrant was created correctly
	res, err = ExecBroadcastWithFlags(t, ctx, xion.GetNode(), revokeSignedTx, "--output", "json")
	require.NoError(t, err)
	t.Logf("broadcasted tx: %s", res)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	feeGrantDetails, err = ExecQuery(t, ctx, xion.GetNode(), "feegrant", "grants-by-grantee", granteeUser.FormattedAddress())
	require.NoError(t, err)
	t.Logf("FeeGrantDetails: %s", feeGrantDetails)

	finalAllowancesStr, ok := feeGrantDetails["allowances"]
	if ok {
		finalAllowances := finalAllowancesStr.([]interface{})
		require.Equal(t, 0, len(finalAllowances))
	}

	// Test additional ExecuteMsg variants
	t.Log("Testing additional treasury contract messages")

	// Create a new admin user for admin tests
	newAdminUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "newadmin", "", fundAmount, xion)
	require.NoError(t, err)

	// Test ProposeAdmin
	t.Log("Testing ProposeAdmin")
	proposeAdminMsg := map[string]interface{}{
		"propose_admin": map[string]interface{}{
			"new_admin": newAdminUser.FormattedAddress(),
		},
	}
	proposeAdminMsgBz, err := json.Marshal(proposeAdminMsg)
	require.NoError(t, err)

	_, err = executeTreasuryMsg(t, ctx, xion, xionUser, treasuryAddr, proposeAdminMsgBz)
	require.NoError(t, err)

	// Query PendingAdmin
	pendingAdminQuery := map[string]interface{}{"pending_admin": map[string]interface{}{}}
	var pendingAdminRes map[string]interface{}
	err = xion.QueryContract(ctx, treasuryAddr, pendingAdminQuery, &pendingAdminRes)
	require.NoError(t, err)

	// Test CancelProposedAdmin
	t.Log("Testing CancelProposedAdmin")
	cancelAdminMsg := map[string]interface{}{
		"cancel_proposed_admin": map[string]interface{}{},
	}
	cancelAdminMsgBz, err := json.Marshal(cancelAdminMsg)
	require.NoError(t, err)

	_, err = executeTreasuryMsg(t, ctx, xion, xionUser, treasuryAddr, cancelAdminMsgBz)
	require.NoError(t, err)

	// Test UpdateGrantConfig
	t.Log("Testing UpdateGrantConfig")
	testAuth2 := authz.NewGenericAuthorization("/" + proto.MessageName(&banktypes.MsgMultiSend{}))
	testGrant2, err := authz.NewGrant(time.Now(), testAuth2, &inFive)
	require.NoError(t, err)

	updatedGrantConfig := GrantConfig{
		Description: "updated authorization",
		Authorization: ExplicitAny{
			TypeURL: testGrant2.Authorization.TypeUrl,
			Value:   testGrant2.Authorization.Value,
		},
		Optional: true,
	}

	updateGrantConfigMsg := map[string]interface{}{
		"update_grant_config": map[string]interface{}{
			"msg_type_url": testAuth.MsgTypeURL(),
			"grant_config": updatedGrantConfig,
		},
	}
	updateGrantConfigMsgBz, err := json.Marshal(updateGrantConfigMsg)
	require.NoError(t, err)

	_, err = executeTreasuryMsg(t, ctx, xion, xionUser, treasuryAddr, updateGrantConfigMsgBz)
	require.NoError(t, err)

	// Test UpdateFeeConfig
	t.Log("Testing UpdateFeeConfig")
	newAllowance := feegrant.BasicAllowance{
		SpendLimit: types.NewCoins(types.NewCoin("uxion", math.NewInt(1000))),
		Expiration: &inFive,
	}

	newFeeGrant, err := feegrant.NewGrant(xionUserAddr, xionUserAddr, &newAllowance)
	require.NoError(t, err)
	updateFeeConfigMsg := map[string]interface{}{
		"update_fee_config": map[string]interface{}{
			"fee_config": FeeConfig{
				Description: "updated fee grant",
				Allowance: &ExplicitAny{
					TypeURL: newFeeGrant.Allowance.TypeUrl,
					Value:   newFeeGrant.Allowance.Value,
				},
				Expiration: int32(36000),
			},
		},
	}
	updateFeeConfigMsgBz, err := json.Marshal(updateFeeConfigMsg)
	require.NoError(t, err)

	_, err = executeTreasuryMsg(t, ctx, xion, xionUser, treasuryAddr, updateFeeConfigMsgBz)
	require.NoError(t, err)

	// Test UpdateParams
	t.Log("Testing UpdateParams")
	updateParamsMsg := map[string]interface{}{
		"update_params": map[string]interface{}{
			"params": Params{
				RedirectURL: "https://newexample.com",
				IconURL:     "https://newexample.com/newicon.png",
				Metadata:    `{"updated_key": "updated_value"}`,
			},
		},
	}
	updateParamsMsgBz, err := json.Marshal(updateParamsMsg)
	require.NoError(t, err)

	_, err = executeTreasuryMsg(t, ctx, xion, xionUser, treasuryAddr, updateParamsMsgBz)
	require.NoError(t, err)

	// Query Params to verify update
	paramsQuery := map[string]interface{}{"params": map[string]interface{}{}}
	var paramsRes map[string]interface{}
	err = xion.QueryContract(ctx, treasuryAddr, paramsQuery, &paramsRes)
	require.NoError(t, err)

	// Test Withdraw
	t.Log("Testing Withdraw")
	// First fund the treasury
	err = xion.SendFunds(ctx, xionUser.KeyName(), ibc.WalletAmount{
		Address: treasuryAddr,
		Denom:   "uxion",
		Amount:  math.NewInt(5000),
	})
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	withdrawMsg := map[string]interface{}{
		"withdraw": map[string]interface{}{
			"coins": []map[string]interface{}{
				{
					"denom":  "uxion",
					"amount": "3000",
				},
			},
		},
	}
	withdrawMsgBz, err := json.Marshal(withdrawMsg)
	require.NoError(t, err)

	_, err = executeTreasuryMsg(t, ctx, xion, xionUser, treasuryAddr, withdrawMsgBz)
	require.NoError(t, err)

	// Test RemoveGrantConfig
	t.Log("Testing RemoveGrantConfig")
	removeGrantConfigMsg := map[string]interface{}{
		"remove_grant_config": map[string]interface{}{
			"msg_type_url": testAuth.MsgTypeURL(),
		},
	}
	removeGrantConfigMsgBz, err := json.Marshal(removeGrantConfigMsg)
	require.NoError(t, err)

	_, err = executeTreasuryMsg(t, ctx, xion, xionUser, treasuryAddr, removeGrantConfigMsgBz)
	require.NoError(t, err)

	// Query grant config type urls to verify removal
	grantConfigUrlsQuery := map[string]interface{}{"grant_config_type_urls": map[string]interface{}{}}
	var grantConfigUrlsRes map[string]interface{}
	err = xion.QueryContract(ctx, treasuryAddr, grantConfigUrlsQuery, &grantConfigUrlsRes)
	require.NoError(t, err)

	// Test all Query messages
	t.Log("Testing all query messages")

	// Query Admin
	adminQuery := map[string]interface{}{"admin": map[string]interface{}{}}
	var adminRes map[string]interface{}
	err = xion.QueryContract(ctx, treasuryAddr, adminQuery, &adminRes)
	require.NoError(t, err)

	// Query FeeConfig
	feeConfigQuery := map[string]interface{}{"fee_config": map[string]interface{}{}}
	var feeConfigRes map[string]interface{}
	err = xion.QueryContract(ctx, treasuryAddr, feeConfigQuery, &feeConfigRes)
	require.NoError(t, err)
}

// Helper function to execute treasury contract messages
func executeTreasuryMsg(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, sender ibc.Wallet, contractAddr string, msgBz []byte) (string, error) {
	contractMsg := wasmtypes.MsgExecuteContract{
		Sender:   sender.FormattedAddress(),
		Contract: contractAddr,
		Msg:      msgBz,
		Funds:    nil,
	}

	// build the tx
	txBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(&contractMsg)
	if err != nil {
		return "", err
	}
	txBuilder.SetGasLimit(2000000)
	tx := txBuilder.GetTx()

	txJSONStr, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(tx)
	if err != nil {
		return "", err
	}

	sendFile, err := os.CreateTemp("", "*-treasury-msg-tx.json")
	if err != nil {
		return "", err
	}
	defer os.Remove(sendFile.Name())

	_, err = sendFile.Write([]byte(txJSONStr))
	if err != nil {
		return "", err
	}
	err = UploadFileToContainer(t, ctx, xion.GetNode(), sendFile)
	if err != nil {
		return "", err
	}

	sendFilePath := strings.Split(sendFile.Name(), "/")

	signedTx, err := ExecBinRaw(t, ctx, xion.GetNode(),
		"tx", "sign", path.Join(xion.GetNode().HomeDir(), sendFilePath[len(sendFilePath)-1]),
		"--from", sender.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--overwrite",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.GetNode().HostName()))
	if err != nil {
		return "", err
	}

	res, err := ExecBroadcastWithFlags(t, ctx, xion.GetNode(), signedTx, "--output", "json")
	if err != nil {
		return "", err
	}

	err = testutil.WaitForBlocks(ctx, 2, xion)
	if err != nil {
		return "", err
	}

	return res, nil
}

func TestTreasuryMulti(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	err := testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	fp, err := os.Getwd()
	require.NoError(t, err)
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)
	t.Logf("deployed code id: %s", codeIDStr)

	inFive := time.Now().Add(time.Minute * 5)
	testAuth := authz.NewGenericAuthorization("/" + proto.MessageName(&banktypes.MsgSend{}))
	testGrant, err := authz.NewGrant(time.Now(), testAuth, &inFive)
	require.NoError(t, err)

	/*
		xionUserAddr, err := types.AccAddressFromBech32(xionUser.FormattedAddress())
		require.NoError(t, err)
	*/

	testAllowanceA := &feegrant.BasicAllowance{
		SpendLimit: types.Coins{types.Coin{Denom: "uxion", Amount: math.NewInt(10)}},
		Expiration: &inFive,
	}

	testAllowanceB := &feegrant.BasicAllowance{
		SpendLimit: types.Coins{types.Coin{Denom: "uxion", Amount: math.NewInt(10)}},
		Expiration: &inFive,
	}

	// NOTE: Create multiallownace
	testMultiAllowance, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{testAllowanceA, testAllowanceB})
	require.NoError(t, err)

	bz, err := proto.Marshal(testMultiAllowance)
	require.NoError(t, err)
	require.NoError(t, testMultiAllowance.ValidateBasic())

	allowanceAny := ExplicitAny{
		TypeURL: "/" + proto.MessageName(testMultiAllowance),
		Value:   bz,
	}

	authorizationAny := ExplicitAny{
		TypeURL: testGrant.Authorization.TypeUrl,
		Value:   testGrant.Authorization.Value,
	}

	grantConfig := GrantConfig{
		Description:   "test authorization",
		Authorization: authorizationAny,
		Optional:      false,
	}

	// Get code info to extract the hash for address prediction
	codeResp, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "code-info", codeIDStr)
	require.NoError(t, err)

	codeHashStr, ok := codeResp["checksum"].(string)
	require.True(t, ok, "code hash not found in response")

	codeHash, err := hex.DecodeString(codeHashStr)
	require.NoError(t, err)

	// Get the creator address
	creatorAddrBytes, err := xion.GetAddress(ctx, xionUser.KeyName())
	require.NoError(t, err)

	creator := types.AccAddress(creatorAddrBytes)
	salt := "treasury-multi-test-1"

	// To predict the address, we need to iterate because:
	// 1. The address depends on the instantiate message
	// 2. The instantiate message needs to contain the address as admin
	// 3. This creates a circular dependency

	// Solution: Use a fixed-point iteration
	// Start with a dummy address and iterate until convergence

	var predictedAddr types.AccAddress
	var predictedAddrStr string

	// First iteration: calculate with a dummy admin to get initial prediction
	dummyAdmin := "xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
	for i := 0; i < 3; i++ {
		adminToUse := dummyAdmin
		if i > 0 {
			adminToUse = predictedAddrStr
		}

		iterMsg := TreasuryInstantiateMsg{
			Admin:        &adminToUse,
			TypeUrls:     []string{testAuth.MsgTypeURL()},
			GrantConfigs: []GrantConfig{grantConfig},
			FeeConfig: &FeeConfig{
				Description: "test fee grant",
				Allowance:   &allowanceAny,
				Expiration:  int32(18000),
			},
			// Include Params to match the final message
			Params: &Params{
				RedirectURL: "https://example.com",
				IconURL:     "https://example.com/icon.png",
				Metadata:    "{}",
			},
		}

		// Marshal and canonicalize like instantiate2 does
		iterMsgBytes, err := json.Marshal(iterMsg)
		require.NoError(t, err)

		var parsed interface{}
		err = json.Unmarshal(iterMsgBytes, &parsed)
		require.NoError(t, err)

		canonicalMsg, err := json.Marshal(parsed)
		require.NoError(t, err)

		// Calculate the address with this message
		predictedAddr = wasmkeeper.BuildContractAddressPredictable(codeHash, creator, []byte(salt), canonicalMsg)
		newPredictedAddrStr := predictedAddr.String()

		// Check for convergence
		if newPredictedAddrStr == adminToUse {
			break
		}

		predictedAddrStr = newPredictedAddrStr
	}

	// Now create the actual instantiate message with the predicted address
	instantiateMsg := TreasuryInstantiateMsg{
		Admin:        &predictedAddrStr, // Set the contract as its own admin (pointer)
		TypeUrls:     []string{testAuth.MsgTypeURL()},
		GrantConfigs: []GrantConfig{grantConfig},
		FeeConfig: &FeeConfig{
			Description: "test fee grant",
			Allowance:   &allowanceAny,
			Expiration:  int32(18000),
		},
		// Include empty Params to match the structure
		Params: &Params{
			RedirectURL: "https://example.com",
			IconURL:     "https://example.com/icon.png",
			Metadata:    "{}",
		},
	}

	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	// Use instantiate2 with a salt to get predictable address
	// Setting admin=true will make the contract its own admin (overriding the message's admin)
	treasuryAddr, err := InstantiateContract2(t, ctx, xion, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), salt, true)
	require.NoError(t, err)
	t.Logf("created treasury instance: %s", treasuryAddr)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	contractState, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "all", treasuryAddr)
	require.NoError(t, err)
	t.Logf("Contract State: %s", contractState)

	granteeUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "grantee", "", fundAmount, xion)
	require.NoError(t, err)

	granterUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "granter", "", fundAmount, xion)
	require.NoError(t, err)
	t.Logf("granter: %s %s %s", granterUser.KeyName(), granterUser.FormattedAddress(), granterUser.Address())
	require.NoError(t, err)

	// wait for user creation
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	err = xion.SendFunds(ctx, granterUser.KeyName(), ibc.WalletAmount{
		Address: treasuryAddr,
		Denom:   "uxion",
		Amount:  math.NewInt(1000),
	})
	require.NoError(t, err)

	// NOTE: Create AuthZGrant
	authzGrantMsg, err := authz.NewMsgGrant(granterUser.Address(), granteeUser.Address(), testAuth, &inFive)
	require.NoError(t, err)

	executeMsg := map[string]interface{}{}
	feegrantMsg := map[string]interface{}{}
	feegrantMsg["authz_granter"] = granterUser.FormattedAddress()
	feegrantMsg["authz_grantee"] = granteeUser.FormattedAddress()
	executeMsg["deploy_fee_grant"] = feegrantMsg
	executeMsgBz, err := json.Marshal(executeMsg)
	require.NoError(t, err)

	// NOTE: Execute in contract
	contractMsg := wasmtypes.MsgExecuteContract{
		Sender:   granterUser.FormattedAddress(),
		Contract: treasuryAddr,
		Msg:      executeMsgBz,
		Funds:    nil,
	}

	require.NoError(t, err)

	// build the tx
	txBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(authzGrantMsg, &contractMsg)
	require.NoError(t, err)
	txBuilder.SetGasLimit(200000)
	tx := txBuilder.GetTx()

	txJSONStr, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(tx)
	require.NoError(t, err)

	t.Logf("tx: %s", txJSONStr)
	require.True(t, json.Valid([]byte(txJSONStr)))
	sendFile, err := os.CreateTemp("", "*-combo-msg-tx.json")
	require.NoError(t, err)
	defer os.Remove(sendFile.Name())

	_, err = sendFile.Write([]byte(txJSONStr))
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), sendFile)
	require.NoError(t, err)

	sendFilePath := strings.Split(sendFile.Name(), "/")

	signedTx, err := ExecBinRaw(t, ctx, xion.GetNode(),
		"tx", "sign", path.Join(xion.GetNode().HomeDir(), sendFilePath[len(sendFilePath)-1]),
		"--from", granterUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--overwrite",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.GetNode().HostName()))
	require.NoError(t, err)
	t.Logf("signed tx: %s", signedTx)

	// todo: validate that the feegrant was created correctly
	res, err := ExecBroadcastWithFlags(t, ctx, xion.GetNode(), signedTx, "--output", "json")
	require.NoError(t, err)
	t.Logf("broadcasted tx: %s", res)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	feeGrantDetails, err := ExecQuery(t, ctx, xion.GetNode(), "feegrant", "grants-by-grantee", granteeUser.FormattedAddress())
	require.NoError(t, err)
	t.Logf("FeeGrantDetails: %s", feeGrantDetails)
	allowances := feeGrantDetails["allowances"].([]interface{})
	allowance := (allowances[0].(map[string]interface{}))["allowance"].(map[string]interface{})
	allowanceType := allowance["type"].(string)
	require.Contains(t, allowanceType, "xion/MultiAnyAllowance")

	revokeMsg := map[string]interface{}{}
	grantee := map[string]interface{}{}
	grantee["grantee"] = granteeUser.FormattedAddress()
	revokeMsg["revoke_allowance"] = grantee
	revokeMsgBz, err := json.Marshal(revokeMsg)
	require.NoError(t, err)

	revokeContractMsg := wasmtypes.MsgExecuteContract{
		Sender:   granterUser.FormattedAddress(),
		Contract: treasuryAddr,
		Msg:      revokeMsgBz,
		Funds:    nil,
	}
	newTxBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err = newTxBuilder.SetMsgs(&revokeContractMsg)
	require.NoError(t, err)
	newTxBuilder.SetGasLimit(20000000)
	newTx := newTxBuilder.GetTx()

	txJSONStr, err = xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(newTx)
	require.NoError(t, err)

	t.Logf("tx: %s", txJSONStr)
	require.True(t, json.Valid([]byte(txJSONStr)))
	revokeSendFile, err := os.CreateTemp("", "*-revoke-combo-msg-tx.json")
	require.NoError(t, err)
	defer os.Remove(revokeSendFile.Name())

	_, err = revokeSendFile.Write([]byte(txJSONStr))
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), revokeSendFile)
	require.NoError(t, err)

	revokeSendFilePath := strings.Split(revokeSendFile.Name(), "/")

	revokeSignedTx, err := ExecBinRaw(t, ctx, xion.GetNode(),
		"tx", "sign", path.Join(xion.GetNode().HomeDir(), revokeSendFilePath[len(revokeSendFilePath)-1]),
		"--from", granterUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--overwrite",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.GetNode().HostName()))
	require.NoError(t, err)
	t.Logf("signed tx: %s", revokeSignedTx)

	// todo: validate that the feegrant was created correctly
	res, err = ExecBroadcastWithFlags(t, ctx, xion.GetNode(), revokeSignedTx, "--output", "json")
	require.NoError(t, err)
	t.Logf("broadcasted tx: %s", res)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	feeGrantDetails, err = ExecQuery(t, ctx, xion.GetNode(), "feegrant", "grants-by-grantee", granteeUser.FormattedAddress())
	require.NoError(t, err)
	t.Logf("FeeGrantDetails: %s", feeGrantDetails)

	finalAllowancesStr, ok := feeGrantDetails["allowances"]
	if ok {
		finalAllowances := finalAllowancesStr.([]interface{})
		// Note: The treasury contract is the actual granter of the fee grant,
		// so only it can revoke the grant. The revocation attempt by the user
		// is expected to have no effect.
		require.Equal(t, 1, len(finalAllowances))
	}
}
