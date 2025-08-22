package jwk_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

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

func setupKeeperForGenesis(t testing.TB) (keeper.Keeper, sdk.Context) {
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

	return k, ctx.Ctx
}

func TestInitGenesis(t *testing.T) {
	k, ctx := setupKeeperForGenesis(t)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Create a genesis state with test data
	genState := types.GenesisState{
		Params: types.Params{
			DeploymentGas: 1000000,
			TimeOffset:    1500,
		},
		AudienceList: []types.Audience{
			{
				Aud:   "audience1",
				Admin: admin,
				Key:   "key1",
			},
			{
				Aud:   "audience2",
				Admin: admin,
				Key:   "key2",
			},
		},
	}

	// Test InitGenesis
	require.NotPanics(t, func() {
		jwk.InitGenesis(ctx, k, genState)
	})

	// Verify params were set
	params := k.GetParams(ctx)
	require.Equal(t, genState.Params, params)

	// Verify audiences were set
	audience1, found := k.GetAudience(ctx, "audience1")
	require.True(t, found)
	require.Equal(t, genState.AudienceList[0], audience1)

	audience2, found := k.GetAudience(ctx, "audience2")
	require.True(t, found)
	require.Equal(t, genState.AudienceList[1], audience2)

	// Verify all audiences
	allAudiences := k.GetAllAudience(ctx)
	require.Len(t, allAudiences, 2)
}

func TestExportGenesis(t *testing.T) {
	k, ctx := setupKeeperForGenesis(t)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up test data
	customParams := types.Params{
		DeploymentGas: 2000000,
		TimeOffset:    2000,
	}
	k.SetParams(ctx, customParams)

	audiences := []types.Audience{
		{
			Aud:   "test-audience-1",
			Admin: admin,
			Key:   "test-key-1",
		},
		{
			Aud:   "test-audience-2",
			Admin: admin,
			Key:   "test-key-2",
		},
	}

	for _, audience := range audiences {
		k.SetAudience(ctx, audience)
	}

	// Test ExportGenesis
	exportedGenesis := jwk.ExportGenesis(ctx, k)
	require.NotNil(t, exportedGenesis)

	// Verify exported params
	require.Equal(t, customParams, exportedGenesis.Params)

	// Verify exported audiences
	require.Len(t, exportedGenesis.AudienceList, 2)
	require.Contains(t, exportedGenesis.AudienceList, audiences[0])
	require.Contains(t, exportedGenesis.AudienceList, audiences[1])
}

func TestGenesisRoundTrip(t *testing.T) {
	k1, ctx1 := setupKeeperForGenesis(t)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Set up initial state
	originalParams := types.Params{
		DeploymentGas: 3000000,
		TimeOffset:    3000,
	}
	k1.SetParams(ctx1, originalParams)

	originalAudiences := []types.Audience{
		{
			Aud:   "round-trip-1",
			Admin: admin,
			Key:   "round-trip-key-1",
		},
		{
			Aud:   "round-trip-2",
			Admin: admin,
			Key:   "round-trip-key-2",
		},
	}

	for _, audience := range originalAudiences {
		k1.SetAudience(ctx1, audience)
	}

	// Export genesis from first keeper
	exportedGenesis := jwk.ExportGenesis(ctx1, k1)

	// Create second keeper and import genesis
	k2, ctx2 := setupKeeperForGenesis(t)
	jwk.InitGenesis(ctx2, k2, *exportedGenesis)

	// Export genesis from second keeper
	reExportedGenesis := jwk.ExportGenesis(ctx2, k2)

	// Verify the round trip preserved all data
	require.Equal(t, exportedGenesis.Params, reExportedGenesis.Params)
	require.Equal(t, len(exportedGenesis.AudienceList), len(reExportedGenesis.AudienceList))

	for _, originalAudience := range exportedGenesis.AudienceList {
		require.Contains(t, reExportedGenesis.AudienceList, originalAudience)
	}
}

func TestGenesisValidation(t *testing.T) {
	validAdmin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	tests := []struct {
		name      string
		genesis   types.GenesisState
		expectErr bool
	}{
		{
			name: "valid genesis",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Admin: validAdmin,
						Aud:   "audience1",
						Key:   "key1",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid admin address",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Admin: "invalid-address",
						Aud:   "audience1",
						Key:   "key1",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "duplicate audience",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				AudienceList: []types.Audience{
					{
						Admin: validAdmin,
						Aud:   "audience1",
						Key:   "key1",
					},
					{
						Admin: validAdmin,
						Aud:   "audience1", // Duplicate
						Key:   "key2",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "empty audience list",
			genesis: types.GenesisState{
				Params:       types.DefaultParams(),
				AudienceList: []types.Audience{},
			},
			expectErr: false,
		},
		{
			name: "invalid params",
			genesis: types.GenesisState{
				Params: types.Params{
					DeploymentGas: 0, // Invalid
					TimeOffset:    30000,
				},
				AudienceList: []types.Audience{},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.genesis.Validate()
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
