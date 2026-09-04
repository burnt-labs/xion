package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestMigrateStoreDisablesRegistrationUntilChainConfiguresAddressHash(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	params, err := app.AbstractAccountKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.AddressDerivationHash = make([]byte, 32)
	require.NoError(t, app.AbstractAccountKeeper.SetParams(ctx, params))

	require.NoError(t, app.AbstractAccountKeeper.Migrator().Migrate2to3(ctx))

	migrated, err := app.AbstractAccountKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Empty(t, migrated.AddressDerivationHash)
	require.False(t, migrated.RegistrationEnabled)
	require.Equal(t, uint64(types.DefaultMaxGas), migrated.MaxGasBefore)
}
