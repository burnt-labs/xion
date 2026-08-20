package abstractaccount_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/abstractaccount/simapp"
	simapptesting "github.com/burnt-labs/xion/x/abstractaccount/simapp/testing"
	"github.com/burnt-labs/xion/x/abstractaccount"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/testdata"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

const (
	mockChainID = "dev-1"
	signMode    = signing.SignMode_SIGN_MODE_DIRECT
)

func anteTerminator(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	return ctx, nil
}

func postTerminator(ctx sdk.Context, _ sdk.Tx, _ bool, _ bool) (sdk.Context, error) {
	return ctx, nil
}

func makeBeforeTxDecorator(app *simapp.SimApp) abstractaccount.BeforeTxDecorator {
	return abstractaccount.NewBeforeTxDecorator(app.AbstractAccountKeeper, app.AccountKeeper, app.TxConfig().SignModeHandler())
}

func makeAfterTxDecorator(app *simapp.SimApp) abstractaccount.AfterTxDecorator {
	return abstractaccount.NewAfterTxDecorator(app.AbstractAccountKeeper)
}

func makeMockAccount(keybase keyring.Keyring, uid string, number uint64) (sdk.AccountI, error) {
	record, _, err := keybase.NewMnemonic(
		uid,
		keyring.English,
		sdk.FullFundraiserPath,
		keyring.DefaultBIP39Passphrase,
		hd.Secp256k1,
	)
	if err != nil {
		return nil, err
	}

	pk, err := record.GetPubKey()
	if err != nil {
		return nil, err
	}

	return authtypes.NewBaseAccount(pk.Address().Bytes(), pk, number, 0), nil
}

type Signer struct {
	keyName        string       // the name of the key in the keyring
	acc            sdk.AccountI // the account corresponding to the address
	overrideAccNum *uint64      // if not nil, will override the account number in the AccountI
	overrideSeq    *uint64      // if not nil, will override the sequence in the AccountI
}

func (s *Signer) AccountNumber() uint64 {
	if s.overrideAccNum != nil {
		return *s.overrideAccNum
	}

	return s.acc.GetAccountNumber()
}

func (s *Signer) Sequence() uint64 {
	if s.overrideSeq != nil {
		return *s.overrideSeq
	}

	return s.acc.GetSequence()
}

// Logics in this function is mostly copied from:
// cosmos/cosmos-sdk/x/auth/ante/testutil_test.go/CreateTestTx
func prepareTx(
	ctx sdk.Context, app *simapp.SimApp, keybase keyring.Keyring,
	msgs []sdk.Msg, signers []Signer, chainID string,
	sign bool,
) (authsigning.Tx, error) {
	txBuilder := app.TxConfig().NewTxBuilder()

	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	// round 1: set empty signature
	sigs := []signing.SignatureV2{}

	for _, signer := range signers {
		sig := signing.SignatureV2{
			PubKey: signer.acc.GetPubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  signMode,
				Signature: nil, // empty
			},
			Sequence: signer.acc.GetSequence(),
		}

		sigs = append(sigs, sig)
	}

	if err := txBuilder.SetSignatures(sigs...); err != nil {
		return nil, err
	}

	// an unsigned tx carries the round-1 empty signature (pubkey set, no
	// signature bytes), mirroring how the SDK builds txs for simulation.
	if !sign {
		return txBuilder.GetTx(), nil
	}

	// round 2: sign the tx
	sigs = []signing.SignatureV2{}

	for _, signer := range signers {
		signerData := authsigning.SignerData{
			Address:       signer.acc.GetAddress().String(),
			ChainID:       chainID,
			AccountNumber: signer.AccountNumber(),
			Sequence:      signer.Sequence(),
			PubKey:        signer.acc.GetPubKey(),
		}

		signBytes, err := authsigning.GetSignBytesAdapter(ctx, app.TxConfig().SignModeHandler(), signMode, signerData, txBuilder.GetTx())
		if err != nil {
			return nil, err
		}

		sigBytes, _, err := keybase.Sign(signer.keyName, signBytes, signMode)
		if err != nil {
			return nil, err
		}

		sig := signing.SignatureV2{
			PubKey: signer.acc.GetPubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  signMode,
				Signature: sigBytes,
			},
			Sequence: signer.Sequence(),
		}

		sigs = append(sigs, sig)
	}

	if err := txBuilder.SetSignatures(sigs...); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}

func storeCodeAndRegisterAccount(
	ctx sdk.Context, app *simapp.SimApp, senderAddr sdk.AccAddress,
	_ []byte, msg interface{}, funds sdk.Coins,
) (*types.AbstractAccount, error) {
	k := app.AbstractAccountKeeper
	msgServer := keeper.NewMsgServerImpl(k)

	// store code
	codeID, _, err := k.ContractKeeper().Create(ctx, senderAddr, testdata.AccountWasm, nil)
	if err != nil {
		return nil, err
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	params.AddressDerivationHash = bytes.Repeat([]byte{0xA5}, sha256.Size)
	params.RegistrationEnabled = true
	if err := k.SetParams(ctx, params); err != nil {
		return nil, err
	}

	// prepare the contract instantiate msg
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// register the account
	res, err := msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: senderAddr.String(),
		CodeID: codeID,
		Msg:    msgBytes,
		Funds:  funds,
		Salt:   []byte("henlo"),
	})
	if err != nil {
		return nil, err
	}

	contractAddr, err := sdk.AccAddressFromBech32(res.Address)
	if err != nil {
		return nil, err
	}

	acc := app.AccountKeeper.GetAccount(ctx, contractAddr)
	if acc == nil {
		return nil, errors.New("account not found")
	}

	abcAcc, ok := acc.(*types.AbstractAccount)
	if !ok {
		return nil, errors.New("account is not an AbstractAccount")
	}

	return abcAcc, nil
}

// Test functions for module.go coverage

func TestAppModuleBasic(t *testing.T) {
	moduleBasic := abstractaccount.AppModuleBasic{}

	// Test IsAppModule and IsOnePerModuleType
	moduleBasic.IsAppModule()
	moduleBasic.IsOnePerModuleType()

	// Test Name
	require.Equal(t, types.ModuleName, moduleBasic.Name())

	// Test RegisterInterfaces
	registry := codectypes.NewInterfaceRegistry()
	moduleBasic.RegisterInterfaces(registry)

	// Test RegisterLegacyAminoCodec
	cdc := codec.NewLegacyAmino()
	moduleBasic.RegisterLegacyAminoCodec(cdc)

	// Test DefaultGenesis
	jsonCodec := codec.NewProtoCodec(registry)
	genesis := moduleBasic.DefaultGenesis(jsonCodec)
	require.NotNil(t, genesis)

	// Test ValidateGenesis with valid genesis
	err := moduleBasic.ValidateGenesis(jsonCodec, nil, genesis)
	require.NoError(t, err)

	// Test ValidateGenesis with invalid JSON
	invalidGenesis := []byte(`{"invalid": "json"}`)
	err = moduleBasic.ValidateGenesis(jsonCodec, nil, invalidGenesis)
	require.Error(t, err)

	// Test RegisterGRPCGatewayRoutes
	ctx := client.Context{}
	mux := runtime.NewServeMux()
	moduleBasic.RegisterGRPCGatewayRoutes(ctx, mux)

	// Test GetTxCmd
	txCmd := moduleBasic.GetTxCmd()
	require.IsType(t, &cobra.Command{}, txCmd)

	// Test GetQueryCmd
	queryCmd := moduleBasic.GetQueryCmd()
	require.IsType(t, &cobra.Command{}, queryCmd)

	// Test RegisterRESTRoutes (deprecated)
	moduleBasic.RegisterRESTRoutes(ctx, nil)
}

func TestAppModule(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)
	ctx := app.NewContext(false)

	k := app.AbstractAccountKeeper
	appModule := abstractaccount.NewAppModule(k)

	// Test ConsensusVersion
	version := appModule.ConsensusVersion()
	require.Equal(t, uint64(3), version)

	// Test InitGenesis with valid genesis
	jsonCodec := app.AppCodec()
	genesisState := types.DefaultGenesisState()
	genesisBytes := jsonCodec.MustMarshalJSON(genesisState)

	validatorUpdates := appModule.InitGenesis(ctx, jsonCodec, genesisBytes)
	require.NotNil(t, validatorUpdates)

	// Test ExportGenesis
	exportedGenesis := appModule.ExportGenesis(ctx, jsonCodec)
	require.NotNil(t, exportedGenesis)
}

func TestAppModuleRegisterServices(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)
	k := app.AbstractAccountKeeper
	appModule := abstractaccount.NewAppModule(k)

	// Test RegisterServices method coverage
	// Since the services are already registered in the mock app,
	// we expect this to panic with a "already registered" error,
	// which proves the method is working correctly

	cfg := module.NewConfigurator(app.AppCodec(), app.MsgServiceRouter(), app.GRPCQueryRouter())

	// This should panic with "already registered" error, which is expected behavior
	require.Panics(t, func() {
		appModule.RegisterServices(cfg)
	})
}

func TestAppModuleInitGenesisInvalid(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)
	ctx := app.NewContext(false)

	k := app.AbstractAccountKeeper
	appModule := abstractaccount.NewAppModule(k)
	jsonCodec := app.AppCodec()

	// Test InitGenesis with invalid genesis that will fail validation
	invalidGenesisState := &types.GenesisState{
		Params: &types.Params{
			AllowAllCodeIDs: false,
			AllowedCodeIDs:  []uint64{},
			MaxGasBefore:    0, // Invalid: should be > 0
			MaxGasAfter:     0, // Invalid: should be > 0
		},
		NextAccountId: 1,
	}
	invalidGenesisBytes := jsonCodec.MustMarshalJSON(invalidGenesisState)

	require.Panics(t, func() {
		appModule.InitGenesis(ctx, jsonCodec, invalidGenesisBytes)
	})
}
