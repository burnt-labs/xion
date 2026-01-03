package types_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func CreateNDkimPubKey(domain string, pubKey string, version types.Version, keyType types.KeyType, count int) []types.DkimPubKey {
	var dkimPubKeys []types.DkimPubKey
	for range count {
		selector := uuid.NewString()
		hash, err := types.ComputePoseidonHash(pubKey)
		if err != nil {
			panic(err)
		}
		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:       domain,
			PubKey:       pubKey,
			PoseidonHash: []byte(hash.String()),
			Selector:     selector,
			Version:      version,
			KeyType:      keyType,
		})
	}
	return dkimPubKeys
}

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
				Params: types.Params{VkeyIdentifier: uint64(1)},
			},
			valid: true,
		},
		{
			desc: "genesis state with empty params",
			genState: &types.GenesisState{
				Params: types.Params{},
			},
			valid: true,
		},
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

func TestDefaultGenesis(t *testing.T) {
	genesis := types.DefaultGenesis()
	require.NotNil(t, genesis)
	require.NotNil(t, genesis.Params)

	// Validate default genesis
	err := genesis.Validate()
	require.NoError(t, err)

	// Check default params
	defaultParams := types.DefaultParams()
	require.Equal(t, defaultParams, genesis.Params)
}

func TestDefaultIndex(t *testing.T) {
	require.Equal(t, uint64(1), types.DefaultIndex)
}
