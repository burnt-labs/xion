package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				AudienceList: []types.Audience{
					{
						Aud:   "0",
						Admin: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
					},
					{
						Aud:   "1",
						Admin: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated audience",
			genState: &types.GenesisState{
				AudienceList: []types.Audience{
					{
						Aud:   "0",
						Admin: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
					},
					{
						Aud:   "0",
						Admin: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
