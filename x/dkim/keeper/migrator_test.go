package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestNewMigrator(t *testing.T) {
	f := SetupTest(t)

	migrator := keeper.NewMigrator(f.k)
	require.NotNil(t, migrator)
}

func TestMigrate1to2(t *testing.T) {
	f := SetupTest(t)
	ctx := f.ctx.WithLogger(log.NewNopLogger())

	// Simulate v1 params without PublicInputIndices.
	oldParams := types.Params{
		VkeyIdentifier:     1,
		MaxPubkeySizeBytes: 2048,
	}
	err := f.k.Params.Set(ctx, oldParams)
	require.NoError(t, err)

	migrator := keeper.NewMigrator(f.k)
	err = migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	newParams, err := f.k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, oldParams.VkeyIdentifier, newParams.VkeyIdentifier)
	require.Equal(t, oldParams.MaxPubkeySizeBytes, newParams.MaxPubkeySizeBytes)
	require.Equal(t, types.DefaultPublicInputIndices(), newParams.PublicInputIndices)
}

func TestMigrate2to3(t *testing.T) {
	f := SetupTest(t)
	ctx := f.ctx.WithLogger(log.NewNopLogger())

	// Simulate v2 params without MinRsaKeyBits (zero value).
	oldParams := types.Params{
		VkeyIdentifier:     1,
		MaxPubkeySizeBytes: 512,
		PublicInputIndices: types.DefaultPublicInputIndices(),
		MinRsaKeyBits:      0,
	}
	err := f.k.Params.Set(ctx, oldParams)
	require.NoError(t, err)

	migrator := keeper.NewMigrator(f.k)
	err = migrator.Migrate2to3(ctx)
	require.NoError(t, err)

	newParams, err := f.k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, types.DefaultMinRSAKeyBits, newParams.MinRsaKeyBits)
	require.Equal(t, oldParams.VkeyIdentifier, newParams.VkeyIdentifier)
	require.Equal(t, oldParams.MaxPubkeySizeBytes, newParams.MaxPubkeySizeBytes)
}
