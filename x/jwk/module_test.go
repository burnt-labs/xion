package jwk_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/jwk"
	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func setupModuleTest(t testing.TB) (jwk.AppModule, sdk.Context, keeper.Keeper) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)

	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	// Create codec
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Create param subspace
	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		paramStore,
	)

	// Initialize with default params
	k.SetParams(ctx.Ctx, types.DefaultParams())

	appModule := jwk.NewAppModule(cdc, k, paramStore)
	return appModule, ctx.Ctx, k
}

func TestJWKModule(t *testing.T) {
	// Create codec
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Test NewAppModuleBasic
	appModuleBasic := jwk.NewAppModuleBasic(cdc)
	require.NotNil(t, appModuleBasic)

	// Test Name
	require.Equal(t, types.ModuleName, appModuleBasic.Name())

	// Test RegisterLegacyAminoCodec
	amino := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		appModuleBasic.RegisterLegacyAminoCodec(amino)
	})

	// Test RegisterInterfaces
	registry := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() {
		appModuleBasic.RegisterInterfaces(registry)
	})

	// Test DefaultGenesis
	genesis := appModuleBasic.DefaultGenesis(cdc)
	require.NotNil(t, genesis)

	// Test ValidateGenesis
	err := appModuleBasic.ValidateGenesis(cdc, nil, genesis)
	require.NoError(t, err)

	// Test ValidateGenesis with invalid data
	invalidGenesis := []byte(`{"invalid": "data"}`)
	err = appModuleBasic.ValidateGenesis(cdc, nil, invalidGenesis)
	require.Error(t, err)

	// Test GetTxCmd
	txCmd := appModuleBasic.GetTxCmd()
	require.NotNil(t, txCmd)

	// Test GetQueryCmd
	queryCmd := appModuleBasic.GetQueryCmd()
	require.NotNil(t, queryCmd)
}

func TestAppModule(t *testing.T) {
	appModule, ctx, k := setupModuleTest(t)

	// Test ConsensusVersion
	require.Equal(t, uint64(2), appModule.ConsensusVersion())

	// Test IsOnePerModuleType and IsAppModule (these just need to not panic)
	require.NotPanics(t, func() {
		appModule.IsOnePerModuleType()
	})
	require.NotPanics(t, func() {
		appModule.IsAppModule()
	})

	// Test InitGenesis
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	genState := types.GenesisState{
		Params: types.DefaultParams(),
		AudienceList: []types.Audience{
			{
				Aud:   "test-audience",
				Admin: admin,
				Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
			},
		},
	}

	genBytes, err := cdc.MarshalJSON(&genState)
	require.NoError(t, err)

	validatorUpdates := appModule.InitGenesis(ctx, cdc, genBytes)
	require.Empty(t, validatorUpdates)

	// Verify the genesis state was set
	audience, found := k.GetAudience(ctx, "test-audience")
	require.True(t, found)
	require.Equal(t, "test-audience", audience.Aud)

	// Test ExportGenesis
	exportedBytes := appModule.ExportGenesis(ctx, cdc)
	require.NotNil(t, exportedBytes)

	var exportedGenState types.GenesisState
	err = cdc.UnmarshalJSON(exportedBytes, &exportedGenState)
	require.NoError(t, err)
	require.Len(t, exportedGenState.AudienceList, 1)
	require.Equal(t, "test-audience", exportedGenState.AudienceList[0].Aud)
}

func TestJWKAppModule(t *testing.T) {
	// Create a basic app module for testing
	appModule := jwk.AppModule{}

	// Test IsOnePerModuleType - just verify it doesn't panic
	require.NotPanics(t, func() {
		appModule.IsOnePerModuleType()
	})

	// Test IsAppModule - just verify it doesn't panic
	require.NotPanics(t, func() {
		appModule.IsAppModule()
	})

	// Test ConsensusVersion
	version := appModule.ConsensusVersion()
	require.Equal(t, uint64(2), version)

	// Note: RegisterServices requires a proper configurator to work,
	// so we skip testing it with nil to avoid panics
}

func TestJWKAppModuleBasicGRPC(t *testing.T) {
	appModuleBasic := jwk.NewAppModuleBasic(nil)
	require.NotNil(t, appModuleBasic)

	// Note: RegisterGRPCGatewayRoutes requires proper setup to avoid panics,
	// so we just test that the function exists and can be called with proper setup.
	// The actual implementation currently does register routes, so nil parameters cause panics.
}

// Additional tests for full module coverage

func TestModuleFunctions(t *testing.T) {
	// Test module creation and basic properties
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	appModuleBasic := jwk.NewAppModuleBasic(cdc)

	// Test basic functions
	require.Equal(t, "jwk", appModuleBasic.Name())

	// Test that these don't panic
	require.NotPanics(t, func() {
		appModuleBasic.RegisterLegacyAminoCodec(codec.NewLegacyAmino())
	})

	require.NotPanics(t, func() {
		reg := codectypes.NewInterfaceRegistry()
		appModuleBasic.RegisterInterfaces(reg)
	})

	// Test genesis functions
	defaultGen := appModuleBasic.DefaultGenesis(cdc)
	require.NotNil(t, defaultGen)

	var txConfig client.TxEncodingConfig
	err := appModuleBasic.ValidateGenesis(cdc, txConfig, defaultGen)
	require.NoError(t, err)

	// Test command functions
	txCmd := appModuleBasic.GetTxCmd()
	require.NotNil(t, txCmd)

	queryCmd := appModuleBasic.GetQueryCmd()
	require.NotNil(t, queryCmd)
}

func TestAppModuleProperties(t *testing.T) {
	appModule, _, _ := setupModuleTest(t)

	// Test module functions
	require.Equal(t, uint64(2), appModule.ConsensusVersion())

	// Test that these don't panic
	require.NotPanics(t, func() {
		appModule.IsOnePerModuleType()
	})

	require.NotPanics(t, func() {
		appModule.IsAppModule()
	})
}
