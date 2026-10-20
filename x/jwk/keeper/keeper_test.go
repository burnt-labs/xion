package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func setupKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient_test")

	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	// Create codec
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	return k, ctx.Ctx
}

func TestKeeperLogger(t *testing.T) {
	k, ctx := setupKeeper(t)

	logger := k.Logger(ctx)
	require.NotNil(t, logger)

	// Logger should be of type log.Logger
	require.Implements(t, (*log.Logger)(nil), logger)
}

func TestKeeperParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Test GetParams
	params := k.GetParams(ctx)
	require.NotNil(t, params)
	require.Equal(t, types.DefaultParams(), params)

	// Test SetParams with custom values
	customParams := types.NewParams(1500, 1000000)
	k.SetParams(ctx, customParams)

	// Verify params were set
	retrievedParams := k.GetParams(ctx)
	require.Equal(t, customParams, retrievedParams)
	require.Equal(t, uint64(1000000), retrievedParams.DeploymentGas)
	require.Equal(t, uint64(1500), retrievedParams.TimeOffset)

	// Test GetTimeOffset
	timeOffset := k.GetTimeOffset(ctx)
	require.Equal(t, uint64(1500), timeOffset)

	// Test GetDeploymentGas
	deploymentGas := k.GetDeploymentGas(ctx)
	require.Equal(t, uint64(1000000), deploymentGas)
}
