package jwk_test

import (
	"testing"

	"github.com/burnt-labs/xion/x/jwk"
	"github.com/burnt-labs/xion/x/jwk/types"
	"github.com/stretchr/testify/require"
	keepertest "jwk/testutil/keeper"
	"jwk/testutil/nullify"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		AudienceList: []types.Audience{
			{
				Aud: "0",
			},
			{
				Aud: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.JwkKeeper(t)
	jwk.InitGenesis(ctx, *k, genesisState)
	got := jwk.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.AudienceList, got.AudienceList)
	// this line is used by starport scaffolding # genesis/test/assert
}
