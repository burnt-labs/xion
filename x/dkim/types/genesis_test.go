package types_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func CreateNDkimPubKey(domain string, pubKey string, version types.Version, keyType types.KeyType, count int) []types.DkimPubKey {
	var dkimPubKeys []types.DkimPubKey
	for i := 0; i < count; i++ {
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
				Params: types.Params{DkimPubkeys: CreateNDkimPubKey("xion.burnt.com", "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB", types.Version_VERSION_DKIM1_UNSPECIFIED, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, 10)},
				// this line is used by starport scaffolding # types/genesis/validField
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
			desc: "genesis state with nil dkim pubkeys",
			genState: &types.GenesisState{
				Params: types.Params{DkimPubkeys: nil},
			},
			valid: true,
		},
		{
			desc: "genesis state with empty dkim pubkeys slice",
			genState: &types.GenesisState{
				Params: types.Params{DkimPubkeys: []types.DkimPubKey{}},
			},
			valid: true,
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

func TestDkimPubKey_Equal(t *testing.T) {
	hash, err := types.ComputePoseidonHash("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB")
	require.NoError(t, err)

	basePubKey := &types.DkimPubKey{
		Domain:       "example.com",
		PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
		Selector:     "selector1",
		Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		PoseidonHash: []byte(hash.String()),
	}

	t.Run("equal to identical copy (pointer)", func(t *testing.T) {
		copy := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte(hash.String()),
		}
		require.True(t, basePubKey.Equal(copy))
	})

	t.Run("equal to identical copy (value)", func(t *testing.T) {
		copy := types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte(hash.String()),
		}
		require.True(t, basePubKey.Equal(copy))
	})

	t.Run("not equal when comparing to nil", func(t *testing.T) {
		require.False(t, basePubKey.Equal(nil))
	})

	t.Run("nil receiver equals nil value", func(t *testing.T) {
		var nilPubKey *types.DkimPubKey
		require.True(t, nilPubKey.Equal(nil))
	})

	t.Run("nil receiver not equal to non-nil value", func(t *testing.T) {
		var nilPubKey *types.DkimPubKey
		require.False(t, nilPubKey.Equal(basePubKey))
	})

	t.Run("non-nil not equal to nil pointer", func(t *testing.T) {
		var nilPtr *types.DkimPubKey
		require.False(t, basePubKey.Equal(nilPtr))
	})

	t.Run("not equal to wrong type", func(t *testing.T) {
		require.False(t, basePubKey.Equal("not a DkimPubKey"))
	})

	t.Run("not equal to different type struct", func(t *testing.T) {
		type OtherStruct struct {
			Domain string
		}
		other := OtherStruct{Domain: "example.com"}
		require.False(t, basePubKey.Equal(other))
	})

	t.Run("not equal when domain differs", func(t *testing.T) {
		different := &types.DkimPubKey{
			Domain:       "different.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte(hash.String()),
		}
		require.False(t, basePubKey.Equal(different))
	})

	t.Run("not equal when pubkey differs", func(t *testing.T) {
		different := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "DIFFERENT_PUBKEY",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte(hash.String()),
		}
		require.False(t, basePubKey.Equal(different))
	})

	t.Run("not equal when selector differs", func(t *testing.T) {
		different := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "different-selector",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte(hash.String()),
		}
		require.False(t, basePubKey.Equal(different))
	})

	t.Run("not equal when version differs", func(t *testing.T) {
		different := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version(99), // Different version
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte(hash.String()),
		}
		require.False(t, basePubKey.Equal(different))
	})

	t.Run("not equal when key type differs", func(t *testing.T) {
		different := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType(99), // Different key type
			PoseidonHash: []byte(hash.String()),
		}
		require.False(t, basePubKey.Equal(different))
	})

	t.Run("not equal when poseidon hash differs", func(t *testing.T) {
		different := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte("different-hash"),
		}
		require.False(t, basePubKey.Equal(different))
	})

	t.Run("equal when both poseidon hashes are nil", func(t *testing.T) {
		a := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: nil,
		}
		b := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: nil,
		}
		require.True(t, a.Equal(b))
	})

	t.Run("equal when both poseidon hashes are empty", func(t *testing.T) {
		a := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte{},
		}
		b := &types.DkimPubKey{
			Domain:       "example.com",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA",
			Selector:     "selector1",
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			PoseidonHash: []byte{},
		}
		require.True(t, a.Equal(b))
	})
}
