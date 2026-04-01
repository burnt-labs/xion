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

func TestMigrate2to3(t *testing.T) {
	f := SetupTest(t)
	ctx := f.ctx.WithLogger(log.NewNopLogger())

	// Persist params without newly-added Groth16 fields.
	oldParams := types.Params{
		MaxVkeySizeBytes: 1000,
		UploadChunkSize:  20,
		UploadChunkGas:   10_000,
	}
	require.NoError(t, f.k.Params.Set(ctx, oldParams))

	migrator := keeper.NewMigrator(f.k)
	require.NoError(t, migrator.Migrate2to3(ctx))

	got, err := f.k.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, types.DefaultMaxGroth16ProofSizeBytes, got.MaxGroth16ProofSizeBytes)
	require.Equal(t, types.DefaultMaxGroth16PublicInputSizeBytes, got.MaxGroth16PublicInputSizeBytes)
	require.Equal(t, types.DefaultMaxUltraHonkProofSizeBytes, got.MaxUltraHonkProofSizeBytes)
	require.Equal(t, types.DefaultMaxUltraHonkPublicInputSizeBytes, got.MaxUltraHonkPublicInputSizeBytes)
	require.Equal(t, oldParams.MaxVkeySizeBytes, got.MaxVkeySizeBytes)
}
