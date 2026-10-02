package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestMigrator(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	t.Run("NewMigrator creates migrator correctly", func(t *testing.T) {
		storeKey := storetypes.NewKVStoreKey(types.StoreKey)
		cdc := app.AppCodec()
		migrator := keeper.NewMigrator(storeKey, cdc)
		require.NotNil(t, migrator)
	})

	t.Run("Migrate1to2 successfully migrates store", func(t *testing.T) {
		migrator := app.AbstractAccountKeeper.Migrator()

		// The migration should succeed
		err := migrator.Migrate1to2(ctx)
		require.NoError(t, err)

		// Verify that params are accessible after migration
		params, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.NotNil(t, params)
	})

	t.Run("Migrate2to3 pauses registration pending configuration", func(t *testing.T) {
		migrator := app.AbstractAccountKeeper.Migrator()

		err := migrator.Migrate2to3(ctx)
		require.NoError(t, err)

		// Registration stays paused until the chain upgrade configures the
		// address derivation hash
		params, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.False(t, params.RegistrationEnabled)
		require.Nil(t, params.AddressDerivationHash)
	})
}

func TestMigrationV2Functions(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	t.Run("v2.MigrateStore works with keeper migrator", func(t *testing.T) {
		// Use the keeper's migrator which has access to the store
		migrator := app.AbstractAccountKeeper.Migrator()

		// Run the migration directly
		err := migrator.Migrate1to2(ctx)
		require.NoError(t, err)

		// Verify params are still accessible
		params, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.NotNil(t, params)
	})
}
