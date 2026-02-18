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
			PoseidonHash: hash.Bytes(),
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
			desc: "valid genesis state with default params",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
			},
			valid: true,
		},
		{
			desc: "valid genesis state with custom vkey identifier",
			genState: &types.GenesisState{
				Params: types.Params{
					VkeyIdentifier:     uint64(42),
					MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
					PublicInputIndices: types.DefaultPublicInputIndices(),
				},
			},
			valid: true,
		},
		{
			desc: "invalid genesis state with empty params",
			genState: &types.GenesisState{
				Params: types.Params{},
			},
			valid: false, // Empty params will have zero min_length which is invalid
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

	// Check default DKIM pubkeys are included
	require.NotEmpty(t, genesis.DkimPubkeys, "default genesis should include DKIM pubkeys")
	defaultDkimPubKeys := types.DefaultDkimPubKeys()
	require.Equal(t, len(defaultDkimPubKeys), len(genesis.DkimPubkeys), "default genesis should have all default DKIM pubkeys")
}

func TestDefaultDkimPubKeys(t *testing.T) {
	dkimPubKeys := types.DefaultDkimPubKeys()
	require.NotEmpty(t, dkimPubKeys)

	// Expected domains and selectors
	expectedRecords := map[string]string{
		"gmail.com":    "20230601",
		"icloud.com":   "1a1hai",
		"outlook.com":  "selector1",
		"proton.me":    "ck677gxvmnehzmitcrhii5zb3q.protonmail",
		"yahoo.com":    "s1024",
		"fastmail.com": "fm2",
	}

	require.Equal(t, len(expectedRecords), len(dkimPubKeys), "should have exactly %d default DKIM records", len(expectedRecords))

	// Verify each expected record is present
	for _, record := range dkimPubKeys {
		expectedSelector, exists := expectedRecords[record.Domain]
		require.True(t, exists, "unexpected domain in default DKIM records: %s", record.Domain)
		require.Equal(t, expectedSelector, record.Selector, "selector mismatch for domain %s", record.Domain)
		require.NotEmpty(t, record.PubKey, "public key should not be empty for domain %s", record.Domain)
		require.Equal(t, types.Version_VERSION_DKIM1_UNSPECIFIED, record.Version, "version should be DKIM1 for domain %s", record.Domain)
		require.Equal(t, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, record.KeyType, "key type should be RSA for domain %s", record.Domain)
		require.NotEmpty(t, record.PoseidonHash, "poseidon hash should not be empty for domain %s", record.Domain)
	}
}

func TestDefaultDkimPubKeysValidation(t *testing.T) {
	// Verify that all default DKIM public keys pass validation
	dkimPubKeys := types.DefaultDkimPubKeys()
	params := types.DefaultParams()

	err := types.ValidateDkimPubKeys(dkimPubKeys, params)
	require.NoError(t, err, "default DKIM public keys should pass validation")

	// Verify each public key can be decoded and parsed, and poseidon hash matches
	for _, record := range dkimPubKeys {
		pubKeyBytes, err := types.DecodePubKeyWithLimit(record.PubKey, params.MaxPubkeySizeBytes)
		require.NoError(t, err, "public key for %s should be decodable", record.Domain)

		_, err = types.ParseRSAPublicKey(pubKeyBytes)
		require.NoError(t, err, "public key for %s should be parseable as RSA", record.Domain)

		// Verify poseidon hash matches computed value
		hash, err := types.ComputePoseidonHash(record.PubKey)
		require.NoError(t, err, "poseidon hash for %s should be computable", record.Domain)
		require.Equal(t, hash.Bytes(), record.PoseidonHash, "poseidon hash mismatch for domain %s", record.Domain)
	}
}

func TestDefaultIndex(t *testing.T) {
	require.Equal(t, uint64(1), types.DefaultIndex)
}

func TestGenesisStateValidateRevokedPubkeys(t *testing.T) {
	// Test validation of revoked pubkeys in genesis state
	validPubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

	t.Run("valid genesis with revoked pubkey", func(t *testing.T) {
		gs := &types.GenesisState{
			Params:         types.DefaultParams(),
			RevokedPubkeys: []string{validPubKey},
		}
		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid genesis with invalid revoked pubkey - bad base64", func(t *testing.T) {
		gs := &types.GenesisState{
			Params:         types.DefaultParams(),
			RevokedPubkeys: []string{"not-valid-base64!!!"},
		}
		err := gs.Validate()
		require.Error(t, err)
	})

	t.Run("invalid genesis with invalid revoked pubkey - not RSA", func(t *testing.T) {
		gs := &types.GenesisState{
			Params:         types.DefaultParams(),
			RevokedPubkeys: []string{"dGVzdA=="}, // valid base64 but not an RSA key
		}
		err := gs.Validate()
		require.Error(t, err)
	})

	t.Run("genesis with empty revoked pubkeys", func(t *testing.T) {
		gs := &types.GenesisState{
			Params:         types.DefaultParams(),
			RevokedPubkeys: []string{},
		}
		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("genesis with MaxPubkeySizeBytes zero uses default", func(t *testing.T) {
		// When MaxPubkeySizeBytes is 0, the code sets it to default
		gs := &types.GenesisState{
			Params: types.Params{
				VkeyIdentifier:     1,
				MaxPubkeySizeBytes: 0, // Will be set to default
				PublicInputIndices: types.DefaultPublicInputIndices(),
			},
			RevokedPubkeys: []string{validPubKey},
		}
		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("genesis with revoked pubkey exceeding size limit", func(t *testing.T) {
		gs := &types.GenesisState{
			Params: types.Params{
				VkeyIdentifier:     1,
				MaxPubkeySizeBytes: 10, // Very small limit
				PublicInputIndices: types.DefaultPublicInputIndices(),
			},
			RevokedPubkeys: []string{validPubKey},
		}
		err := gs.Validate()
		require.Error(t, err)
	})
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
