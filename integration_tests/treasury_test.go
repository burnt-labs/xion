package integration_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7/ibc"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xionapp "github.com/burnt-labs/xion/app"
	jwktypes "github.com/burnt-labs/xion/x/jwk/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegranttypes "github.com/cosmos/cosmos-sdk/x/feegrant"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

type GrantConfig struct {
	Description   string       `json:"description"`
	Authorization ExplicitAny  `json:"authorization"`
	Allowance     *ExplicitAny `json:"allowance"`
}

//func (g *GrantConfig) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
//	var allowance feegranttypes.FeeAllowanceI
//	if err := unpacker.UnpackAny(g.Allowance, &allowance); err != nil {
//		return err
//	}
//	var authorization authztypes.Authorization
//	return unpacker.UnpackAny(g.Authorization, &authorization)
//}
//
//func (t *TreasuryInstantiateMsg) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
//	for _, g := range t.GrantConfigs {
//		if err := codectypes.UnpackInterfaces(g, unpacker); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//var _ codectypes.UnpackInterfacesMessage = (*GrantConfig)(nil)
//var _ codectypes.UnpackInterfacesMessage = (*TreasuryInstantiateMsg)(nil)

type TreasuryInstantiateMsg struct {
	Admin        types.AccAddress `json:"admin"`
	TypeUrls     []string         `json:"type_urls"`
	GrantConfigs []GrantConfig    `json:"grant_configs"`
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
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	err := testutil.WaitForBlocks(ctx, 8, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// register any needed msg types
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.Msg)(nil),
		&xiontypes.MsgSetPlatformPercentage{},
		&xiontypes.MsgSend{},
		&wasmtypes.MsgInstantiateContract{},
		&wasmtypes.MsgExecuteContract{},
		&wasmtypes.MsgStoreCode{},
		&aatypes.MsgUpdateParams{},
		&aatypes.MsgRegisterAccount{},
		&jwktypes.MsgCreateAudience{},
		&authztypes.MsgGrant{},
	)
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*authtypes.AccountI)(nil), &aatypes.AbstractAccount{})
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*cryptotypes.PubKey)(nil), &aatypes.NilPubKey{})

	xion.Config().EncodingConfig.InterfaceRegistry.RegisterInterface(
		"cosmos.feegrant.v1beta1.FeeAllowanceI",
		(*feegranttypes.FeeAllowanceI)(nil),
		&feegranttypes.BasicAllowance{},
	)

	xion.Config().EncodingConfig.InterfaceRegistry.RegisterInterface(
		"cosmos.authz.v1beta1.Authorization",
		(*authztypes.Authorization)(nil),
		&authztypes.GenericAuthorization{},
	)

	fp, err := os.Getwd()
	require.NoError(t, err)
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)
	t.Logf("deployed code id: %s", codeIDStr)

	inFive := time.Now().Add(time.Minute * 5)
	testAuth := authz.NewGenericAuthorization("/cosmos.bank.v1beta1/MsgSend")
	testGrant, err := authz.NewGrant(inFive, testAuth, nil)
	require.NoError(t, err)

	xionUserAddr, err := types.AccAddressFromBech32(xionUser.FormattedAddress())
	require.NoError(t, err)

	testAllowance := feegrant.BasicAllowance{
		SpendLimit: types.Coins{},
		Expiration: &inFive,
	}
	feeGrant, err := feegrant.NewGrant(xionUserAddr, xionUserAddr, &testAllowance)
	require.NoError(t, err)

	authorizationAny := ExplicitAny{
		TypeURL: testGrant.Authorization.TypeUrl,
		Value:   testGrant.Authorization.Value,
	}
	allowanceAny := ExplicitAny{
		TypeURL: feeGrant.Allowance.TypeUrl,
		Value:   feeGrant.Allowance.Value,
	}
	grantConfig := GrantConfig{
		Description:   "test authorization",
		Authorization: authorizationAny,
		Allowance:     &allowanceAny,
	}
	authorizationStr, err := json.Marshal(authorizationAny)
	require.NoError(t, err)
	t.Logf("authorization: %s", authorizationStr)

	instantiateMsg := TreasuryInstantiateMsg{
		TypeUrls:     []string{testAuth.Msg},
		GrantConfigs: []GrantConfig{grantConfig},
	}

	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	treasuryAddr, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
	require.NoError(t, err)
	t.Logf("created treasury instance: %s", treasuryAddr)

	// todo: create a tx that grants an authz grant for the same authorization
	// as well as a call to this contract requesting a feegrant
	// validate the feegrant exists on chain
	granteeUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "grantee", "", fundAmount, xion)
	require.NoError(t, err)

	granterUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "granter", "", fundAmount, xion)
	require.NoError(t, err)
	t.Logf("granter: %s %s %s", granterUser.KeyName(), granterUser.FormattedAddress(), granterUser.Address())
	require.NoError(t, err)

	// wait for user creation
	err = testutil.WaitForBlocks(ctx, 4, xion)
	require.NoError(t, err)

	err = xion.SendFunds(ctx, granterUser.KeyName(), ibc.WalletAmount{
		Address: treasuryAddr,
		Denom:   "uxion",
		Amount:  1000,
	})
	require.NoError(t, err)

	authzGrantMsg, err := authz.NewMsgGrant(granterUser.Address(), granteeUser.Address(), testAuth, &inFive)
	require.NoError(t, err)
	encodingConfig := xionapp.MakeEncodingConfig()

	executeMsg := map[string]interface{}{}
	feegrantMsg := map[string]interface{}{}
	feegrantMsg["authz_granter"] = granterUser.FormattedAddress()
	feegrantMsg["authz_grantee"] = granteeUser.FormattedAddress()
	feegrantMsg["msg_type_url"] = testAuth.Msg
	executeMsg["deploy_fee_grant"] = feegrantMsg
	executeMsgBz, err := json.Marshal(executeMsg)
	require.NoError(t, err)

	contractMsg := wasmtypes.MsgExecuteContract{
		Sender:   granterUser.FormattedAddress(),
		Contract: treasuryAddr,
		Msg:      executeMsgBz,
		Funds:    nil,
	}

	// build the tx
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(authzGrantMsg, &contractMsg)
	require.NoError(t, err)
	txBuilder.SetGasLimit(200000)
	tx := txBuilder.GetTx()

	txJSONStr, err := encodingConfig.TxConfig.TxJSONEncoder()(tx)
	require.NoError(t, err)

	t.Logf("tx: %s", txJSONStr)
	require.True(t, json.Valid([]byte(txJSONStr)))
	sendFile, err := os.CreateTemp("", "*-combo-msg-tx.json")
	require.NoError(t, err)
	defer os.Remove(sendFile.Name())

	_, err = sendFile.Write([]byte(txJSONStr))
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.FullNodes[0], sendFile)
	require.NoError(t, err)

	sendFilePath := strings.Split(sendFile.Name(), "/")

	signedTx, err := ExecBinRaw(t, ctx, xion.FullNodes[0],
		"tx", "sign", path.Join(xion.FullNodes[0].HomeDir(), sendFilePath[len(sendFilePath)-1]),
		"--from", granterUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--overwrite",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.FullNodes[0].HostName()))
	require.NoError(t, err)
	t.Logf("signed tx: %s", signedTx)

	// todo: validate that the feegrant was created correctly
	output, err := ExecBroadcast(t, ctx, xion.FullNodes[0], signedTx)
	require.NoError(t, err)
	t.Logf("broadcasted tx: %s", output)

	var outputJSON map[string]string
	var txOutputJSON map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &outputJSON))
	require.NoError(t, json.Unmarshal([]byte(outputJSON["tx"]), &txOutputJSON))

	txDetails, err := ExecQuery(t, ctx, xion.FullNodes[0], "tx", txOutputJSON["txhash"])
	t.Logf("TxDetails: %s", txDetails)
	require.NoError(t, err)
}
