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
		{
			desc: "genesis state with zero MaxPubkeySizeBytes gets default",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxPubkeySizeBytes: 0,
					VkeyIdentifier:     1,
				},
			},
			valid: true,
		},
		{
			desc: "genesis state with invalid dkim pubkeys validates params error",
			genState: &types.GenesisState{
				Params: types.Params{
					MaxPubkeySizeBytes: 0,
					VkeyIdentifier:     0,
				},
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "test.com",
						Selector: "sel",
						PubKey:   "validRSAPubKey",
					},
				},
			},
			valid: false,
		},
		{
			desc: "genesis state with invalid dkim pubkey",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:   "test.com",
						Selector: "selector",
						PubKey:   "invalid_base64",
						Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
						KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
					},
				},
			},
			valid: false,
		},
		{
			desc: "genesis state with invalid revoked pubkey",
			genState: &types.GenesisState{
				Params:          types.DefaultParams(),
				RevokedPubkeys: []string{"invalid_base64"},
			},
			valid: false,
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

func TestDkimPubKeyEqual(t *testing.T) {
	base := &types.DkimPubKey{
		Domain:       "example.com",
		PubKey:       "pub",
		PoseidonHash: []byte{1, 2, 3},
		Selector:     "selector",
		Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
	}

	t.Run("matches struct value", func(t *testing.T) {
		copy := *base
		require.True(t, base.Equal(copy))
	})

	t.Run("different field", func(t *testing.T) {
		other := *base
		other.Domain = "other.com"
		require.False(t, base.Equal(other))
	})

	t.Run("nil comparisons", func(t *testing.T) {
		var nilKey *types.DkimPubKey
		require.True(t, nilKey.Equal(nil))
		require.False(t, nilKey.Equal(base))
	})

	t.Run("wrong type", func(t *testing.T) {
		require.False(t, base.Equal("not-a-dkim-key"))
	})
}
