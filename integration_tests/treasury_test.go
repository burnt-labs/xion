package integration_tests

import (
	"encoding/json"
	"fmt"
	xionapp "github.com/burnt-labs/xion/app"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	jwktypes "github.com/burnt-labs/xion/x/jwk/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
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
		&wasmtypes.MsgStoreCode{},
		&aatypes.MsgUpdateParams{},
		&aatypes.MsgRegisterAccount{},
		&jwktypes.MsgCreateAudience{},
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
		TypeUrls:     []string{testGrant.Authorization.TypeUrl},
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
	granteeKey := "grantee"
	err = xion.CreateKey(ctx, granteeKey)
	require.NoError(t, err)
	granteeAddressBytes, err := xion.GetAddress(ctx, granteeKey)
	require.NoError(t, err)
	granteeAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, granteeAddressBytes)
	require.NoError(t, err)

	granterKey := "granter"
	err = xion.CreateKey(ctx, granterKey)
	require.NoError(t, err)
	granterAddressBytes, err := xion.GetAddress(ctx, granterKey)
	require.NoError(t, err)
	granterAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, granterAddressBytes)
	require.NoError(t, err)

	//genericAuthzAuth := authz.GenericAuthorization{
	//	Msg: "cosmos-sdk/MsgSend",
	//}
	//authzAny, err := cdctypes.NewAnyWithValue(&genericAuthzAuth)
	//require.NoError(t, err)

	//	authzGrantMsgStr := fmt.Sprintf(`
	//{
	//"@type":"/cosmos.authz.v1beta1.MsgGrant",
	//"grant":
	//}`)

	authzGrantMsg, err := authz.NewMsgGrant(types.AccAddress(granterAddress), types.AccAddress(granteeAddress), testAuth, &inFive)
	require.NoError(t, err)
	encodingConfig := xionapp.MakeEncodingConfig()
	authzGrantMsgStr, err := encodingConfig.Marshaler.MarshalInterfaceJSON(authzGrantMsg)
	require.NoError(t, err)

	executeMsg := map[string]interface{}{}
	feegrantMsg := map[string]interface{}{}
	feegrantMsg["authz_granter"] = granterAddress
	feegrantMsg["authz_grantee"] = granteeAddress
	feegrantMsg["authorization"] = authorizationAny
	executeMsg["deploy_fee_grant"] = feegrantMsg
	executeMsgBz, err := json.Marshal(executeMsg)
	require.NoError(t, err)

	contractMsg := wasmtypes.MsgExecuteContract{
		Sender:   granterAddress,
		Contract: treasuryAddr,
		Msg:      executeMsgBz,
		Funds:    nil,
	}
	contractMsgStr, err := encodingConfig.Marshaler.MarshalInterfaceJSON(&contractMsg)
	require.NoError(t, err)

	txJSONStr := fmt.Sprintf(`
		{
		 "body": {
		   "messages": [
		     %s,
			 %s
		   ],
		   "memo": "",
		   "timeout_height": "0",
		   "extension_options": [],
		   "non_critical_extension_options": []
		 },
		 "auth_info": {
		   "signer_infos": [],
		   "fee": {
		     "amount": [],
		     "gas_limit": "200000",
		     "payer": "",
		     "granter": ""
		   },
		   "tip": null
		 },
		 "signatures": []
		}
			`, authzGrantMsgStr, contractMsgStr)
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
	signedTx, err := ExecBin(t, ctx, xion.FullNodes[0], "tx", "sign", "--from", granterKey, path.Join(xion.FullNodes[0].HomeDir(), sendFilePath[len(sendFilePath)-1]))
	require.NoError(t, err)
	t.Logf("signed tx: %s", signedTx)

	//tx, err := encodingConfig.TxConfig.TxJSONDecoder()([]byte(txJSONStr))
	//require.NoError(t, err)

}
