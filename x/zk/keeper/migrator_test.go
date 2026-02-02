package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	"github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
)

func TestNewMigrator(t *testing.T) {
	f := SetupTest(t)

	migrator := keeper.NewMigrator(f.k)
	require.NotNil(t, migrator)
}

func TestMigrate1to2(t *testing.T) {
	f := SetupTest(t)
	ctx := f.ctx.WithLogger(log.NewNopLogger())

	// Simulate v1 vkeys state with old vkey.
	oldVkey := types.VKey{}
	err := f.k.VKeys.Set(ctx, 1, oldVkey)
	require.NoError(t, err)

	migrator := keeper.NewMigrator(f.k)
	err = migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	newVkey, err := f.k.VKeys.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, newVkey.KeyBytes, types.CreateDefaultVKeyBytes())
}
