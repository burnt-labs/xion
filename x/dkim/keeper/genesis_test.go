package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestGenesis(t *testing.T) {
	f := SetupTest(t)

	genesisState := &types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	err := f.k.InitGenesis(f.ctx, genesisState)
	if err != nil {
		panic(err)
	}

	got := f.k.ExportGenesis(f.ctx)
	require.NotNil(t, got)

	// this line is used by starport scaffolding # genesis/test/assert
}
