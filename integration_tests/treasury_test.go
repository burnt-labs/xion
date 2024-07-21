package integration_tests

import (
	"cosmossdk.io/math"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	"cosmossdk.io/x/feegrant"
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
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

type GrantConfig struct {
	Description   string                 `json:"description"`
	Authorization map[string]interface{} `json:"authorization"`
	Allowance     *ExplicitAny           `json:"allowance"`
}

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
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	err := testutil.WaitForBlocks(ctx, 2, xion)
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
		(*feegrant.FeeAllowanceI)(nil),
		&feegrant.BasicAllowance{},
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
	testAuth := authz.NewGenericAuthorization("/" + proto.MessageName(&banktypes.MsgSend{}))
	testGrant, err := authz.NewGrant(time.Now(), testAuth, &inFive)
	require.NoError(t, err)

	xionUserAddr, err := types.AccAddressFromBech32(xionUser.FormattedAddress())
	require.NoError(t, err)

	testAllowance := feegrant.BasicAllowance{
		SpendLimit: types.Coins{},
		Expiration: &inFive,
	}
	feeGrant, err := feegrant.NewGrant(xionUserAddr, xionUserAddr, &testAllowance)
	require.NoError(t, err)

	allowanceAny := ExplicitAny{
		TypeURL: feeGrant.Allowance.TypeUrl,
		Value:   feeGrant.Allowance.Value,
	}

	authorizationAny := map[string]interface{}{}
	authorizationAny["@type"] = testGrant.Authorization.TypeUrl
	authorizationAny["msg"] = testAuth.MsgTypeURL()

	grantConfig := GrantConfig{
		Description:   "test authorization",
		Authorization: authorizationAny,
		Allowance:     &allowanceAny,
	}

	instantiateMsg := TreasuryInstantiateMsg{
		TypeUrls:     []string{testAuth.MsgTypeURL()},
		GrantConfigs: []GrantConfig{grantConfig},
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

	authzGrantMsg, err := authz.NewMsgGrant(granterUser.Address(), granteeUser.Address(), testAuth, &inFive)
	require.NoError(t, err)
	encodingConfig := xionapp.MakeEncodingConfig(t)

	executeMsg := map[string]interface{}{}
	feegrantMsg := map[string]interface{}{}
	feegrantMsg["authz_granter"] = granterUser.FormattedAddress()
	feegrantMsg["authz_grantee"] = granteeUser.FormattedAddress()
	feegrantMsg["msg_type_url"] = testAuth.MsgTypeURL()
	executeMsg["deploy_fee_grant"] = feegrantMsg
	executeMsgBz, err := json.Marshal(executeMsg)
	require.NoError(t, err)

	contractMsg := wasmtypes.MsgExecuteContract{
		Sender:   granterUser.FormattedAddress(),
		Contract: treasuryAddr,
		Msg:      executeMsgBz,
		Funds:    nil,
	}

	require.NoError(t, err)
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
	allowanceType := allowance["@type"].(string)
	require.Contains(t, allowanceType, "/cosmos.feegrant.v1beta1.BasicAllowance")
}
