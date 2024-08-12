package integration_tests

import (
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
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
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
	Admin        types.AccAddress `json:"admin,omitempty"`
	TypeUrls     []string         `json:"type_urls"`
	GrantConfigs []GrantConfig    `json:"grant_configs"`
	FeeConfig    *FeeConfig       `json:"fee_config"`
}

type ExplicitAny struct {
	TypeURL string `json:"type_url"`
	Value   []byte `json:"value"`
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

	// NOTE: Start the Treasury
	instantiateMsg := TreasuryInstantiateMsg{
		TypeUrls:     []string{testAuth.MsgTypeURL()},
		GrantConfigs: []GrantConfig{grantConfig},
		FeeConfig: &FeeConfig{
			Description: "test fee grant",
			Allowance:   &allowanceAny,
			Expiration:  int32(18000),
		},
	}

	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	treasuryAddr, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
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

	txDetails, err := ExecQuery(t, ctx, xion.GetNode(), "tx", res)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)

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

	txDetails, err = ExecQuery(t, ctx, xion.GetNode(), "tx", res)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)

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

	// NOTE: Start the Treasury
	instantiateMsg := TreasuryInstantiateMsg{
		TypeUrls:     []string{testAuth.MsgTypeURL()},
		GrantConfigs: []GrantConfig{grantConfig},
		FeeConfig: &FeeConfig{
			Description: "test fee grant",
			Allowance:   &allowanceAny,
			Expiration:  int32(18000),
		},
	}

	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	treasuryAddr, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
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

	txDetails, err := ExecQuery(t, ctx, xion.GetNode(), "tx", res)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)

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

	txDetails, err = ExecQuery(t, ctx, xion.GetNode(), "tx", res)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)

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
}
