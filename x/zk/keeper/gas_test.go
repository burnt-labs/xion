package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
)

func TestConsumeVKeyGas(t *testing.T) {
	t.Run("consumes expected gas for payload size", func(t *testing.T) {
		f := SetupTest(t)
		ctx := f.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000))

		params := types.NewParams(100, 20, 10)
		err := keeper.ConsumeVKeyGas(ctx, params, 25)
		require.NoError(t, err)
		require.Equal(t, uint64(20), ctx.GasMeter().GasConsumed())
	})

	t.Run("returns error for invalid size without consuming gas", func(t *testing.T) {
		f := SetupTest(t)
		ctx := f.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000))

		err := keeper.ConsumeVKeyGas(ctx, types.DefaultParams(), 0)
		require.Error(t, err)
		require.Equal(t, uint64(0), ctx.GasMeter().GasConsumed())
	})
}
