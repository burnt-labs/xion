package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestInitGenesis(t *testing.T) {
	keeper, ctx := setupKeeperForTesting(t)

	// Test case 1: Valid genesis state
	t.Run("valid genesis state", func(t *testing.T) {
		platformPercentage := uint32(10)
		platformMinimums := sdk.NewCoins(
			sdk.NewCoin("utoken", math.NewInt(1000)),
			sdk.NewCoin("uatom", math.NewInt(2000)),
		)

		genesisState := &types.GenesisState{
			PlatformPercentage: platformPercentage,
			PlatformMinimums:   platformMinimums,
		}

		// Call InitGenesis
		keeper.InitGenesis(ctx, genesisState)

		// Verify that the values were set correctly
		storedPercentage := keeper.GetPlatformPercentage(ctx)
		require.True(t, math.NewIntFromUint64(uint64(platformPercentage)).Equal(storedPercentage))

		storedMinimums, err := keeper.GetPlatformMinimums(ctx)
		require.NoError(t, err)
		require.Equal(t, len(platformMinimums), len(storedMinimums))

		for i, expected := range platformMinimums {
			require.Equal(t, expected.Denom, storedMinimums[i].Denom)
			require.True(t, expected.Amount.Equal(storedMinimums[i].Amount))
		}
	})

	// Test case 2: Empty platform minimums
	t.Run("empty platform minimums", func(t *testing.T) {
		platformPercentage := uint32(5)
		platformMinimums := sdk.NewCoins()

		genesisState := &types.GenesisState{
			PlatformPercentage: platformPercentage,
			PlatformMinimums:   platformMinimums,
		}

		// Call InitGenesis
		keeper.InitGenesis(ctx, genesisState)

		// Verify that the percentage was set and minimums are empty
		storedPercentage := keeper.GetPlatformPercentage(ctx)
		require.True(t, math.NewIntFromUint64(uint64(platformPercentage)).Equal(storedPercentage))

		storedMinimums, err := keeper.GetPlatformMinimums(ctx)
		require.NoError(t, err)
		require.Empty(t, storedMinimums)
	})

	// Test case 3: Zero platform percentage
	t.Run("zero platform percentage", func(t *testing.T) {
		platformPercentage := uint32(0)
		platformMinimums := sdk.NewCoins(
			sdk.NewCoin("utest", math.NewInt(500)),
		)

		genesisState := &types.GenesisState{
			PlatformPercentage: platformPercentage,
			PlatformMinimums:   platformMinimums,
		}

		// Call InitGenesis
		keeper.InitGenesis(ctx, genesisState)

		// Verify values
		storedPercentage := keeper.GetPlatformPercentage(ctx)
		require.True(t, math.NewIntFromUint64(uint64(platformPercentage)).Equal(storedPercentage))

		storedMinimums, err := keeper.GetPlatformMinimums(ctx)
		require.NoError(t, err)
		require.Len(t, storedMinimums, 1)
		require.Equal(t, "utest", storedMinimums[0].Denom)
		require.True(t, math.NewInt(500).Equal(storedMinimums[0].Amount))
	})
}

func TestExportGenesis(t *testing.T) {
	keeper, ctx := setupKeeperForTesting(t)

	// Test case 1: Test with initialized values first to avoid the empty data panic
	t.Run("with initialized values", func(t *testing.T) {
		// Set some initial value to avoid empty data panic
		keeper.OverwritePlatformPercentage(ctx, 0)
		platformMinimums := sdk.NewCoins(
			sdk.NewCoin("ubtc", math.NewInt(100)),
		)
		err := keeper.OverwritePlatformMinimum(ctx, platformMinimums)
		require.NoError(t, err)

		exportedGenesis := keeper.ExportGenesis(ctx)

		// Due to the uint32/uint64 bug, percentage will be 0
		require.Equal(t, uint32(0), exportedGenesis.PlatformPercentage)
		require.Equal(t, len(platformMinimums), len(exportedGenesis.PlatformMinimums))

		for i, expected := range platformMinimums {
			require.Equal(t, expected.Denom, exportedGenesis.PlatformMinimums[i].Denom)
			require.True(t, expected.Amount.Equal(exportedGenesis.PlatformMinimums[i].Amount))
		}
	})

	// Test case 2: Verify that percentage export now works correctly after fix
	t.Run("percentage export works correctly", func(t *testing.T) {
		platformPercentage := uint32(15)
		keeper.OverwritePlatformPercentage(ctx, platformPercentage)

		// Verify storage works correctly
		storedPercentage := keeper.GetPlatformPercentage(ctx)
		require.True(t, math.NewIntFromUint64(uint64(platformPercentage)).Equal(storedPercentage))

		// Export now correctly returns the value after the fix
		exportedGenesis := keeper.ExportGenesis(ctx)

		// Verify that ExportGenesis now correctly reads the platform percentage
		require.Equal(t, platformPercentage, exportedGenesis.PlatformPercentage)
	})
}

func TestGenesisRoundTrip(t *testing.T) {
	// Test with only minimums
	t.Run("minimums only round trip", func(t *testing.T) {
		keeper1, ctx1 := setupKeeperForTesting(t)

		// Create an initial genesis state with only minimums
		originalGenesis := &types.GenesisState{
			PlatformPercentage: uint32(0), // Zero percentage for this test
			PlatformMinimums: sdk.NewCoins(
				sdk.NewCoin("uround", math.NewInt(999)),
				sdk.NewCoin("utrip", math.NewInt(777)),
			),
		}

		// Initialize with the original genesis
		keeper1.InitGenesis(ctx1, originalGenesis)

		// Export the genesis state
		exportedGenesis := keeper1.ExportGenesis(ctx1)

		// Verify that exported genesis matches original for minimums
		require.Equal(t, originalGenesis.PlatformPercentage, exportedGenesis.PlatformPercentage)
		require.Equal(t, len(originalGenesis.PlatformMinimums), len(exportedGenesis.PlatformMinimums))

		for i, original := range originalGenesis.PlatformMinimums {
			exported := exportedGenesis.PlatformMinimums[i]
			require.Equal(t, original.Denom, exported.Denom)
			require.True(t, original.Amount.Equal(exported.Amount))
		}

		// Initialize a new keeper with the exported genesis and verify it works
		keeper2, ctx2 := setupKeeperForTesting(t)
		keeper2.InitGenesis(ctx2, exportedGenesis)

		// Verify the second keeper has the same state
		storedPercentage := keeper2.GetPlatformPercentage(ctx2)
		require.True(t, math.NewIntFromUint64(uint64(originalGenesis.PlatformPercentage)).Equal(storedPercentage))

		storedMinimums, err := keeper2.GetPlatformMinimums(ctx2)
		require.NoError(t, err)
		require.Equal(t, len(originalGenesis.PlatformMinimums), len(storedMinimums))
	})

	// Test percentage round-trip works correctly after fix
	t.Run("percentage round trip works", func(t *testing.T) {
		keeper1, ctx1 := setupKeeperForTesting(t)

		// Try with a non-zero percentage
		originalGenesis := &types.GenesisState{
			PlatformPercentage: uint32(25),
			PlatformMinimums:   sdk.NewCoins(),
		}

		// Initialize with the original genesis
		keeper1.InitGenesis(ctx1, originalGenesis)

		// Verify the percentage was set correctly in storage
		storedPercentage := keeper1.GetPlatformPercentage(ctx1)
		require.True(t, math.NewIntFromUint64(uint64(originalGenesis.PlatformPercentage)).Equal(storedPercentage))

		// Export the genesis state
		exportedGenesis := keeper1.ExportGenesis(ctx1)

		// After the fix, percentage is correctly preserved in export
		require.Equal(t, originalGenesis.PlatformPercentage, exportedGenesis.PlatformPercentage)
	})
}

// Helper function to set up keeper for testing
func setupKeeperForTesting(t *testing.T) (*Keeper, sdk.Context) {
	t.Helper()

	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	keeper := Keeper{
		storeKey: key,
	}

	return &keeper, ctx
}
