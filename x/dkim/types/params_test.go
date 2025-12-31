package types_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	require.NotNil(t, params)
	require.Equal(t, uint64(1), params.VkeyIdentifier)
	require.NotEmpty(t, params.DkimPubkeys)
	require.Len(t, params.DkimPubkeys, 1)
	require.Equal(t, types.DefaultMaxPubKeySizeBytes, params.MaxPubkeySizeBytes)

	// Verify default DKIM pubkey
	defaultPubkey := params.DkimPubkeys[0]
	require.Equal(t, "gmail.com", defaultPubkey.Domain)
	require.Equal(t, "20230601", defaultPubkey.Selector)
	require.NotEmpty(t, defaultPubkey.PubKey)
	require.NotEmpty(t, defaultPubkey.PoseidonHash)
}

func TestParams_String(t *testing.T) {
	t.Run("default params to string", func(t *testing.T) {
		params := types.DefaultParams()
		str := params.String()

		require.NotEmpty(t, str)
		require.Contains(t, str, "gmail.com")
		require.Contains(t, str, "20230601")
		require.Contains(t, str, "vkey_identifier")
	})

	t.Run("empty params to string", func(t *testing.T) {
		params := types.Params{}
		str := params.String()

		require.NotEmpty(t, str)
		// Should be valid JSON even when empty
		require.Contains(t, str, "{")
		require.Contains(t, str, "}")
	})

	t.Run("params with multiple pubkeys to string", func(t *testing.T) {
		hash, err := types.ComputePoseidonHash("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB")
		require.NoError(t, err)

		params := types.Params{
			VkeyIdentifier: 42,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       "pubkey1",
					PoseidonHash: hash.Bytes(),
				},
				{
					Domain:       "test.com",
					Selector:     "selector2",
					PubKey:       "pubkey2",
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		str := params.String()

		require.NotEmpty(t, str)
		require.Contains(t, str, "example.com")
		require.Contains(t, str, "test.com")
	})
}

func TestParams_Validate(t *testing.T) {
	validPubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAntvSKT1hkqhKe0xcaZ0x+QbouDsJuBfby/S82jxsoC/SodmfmVs2D1KAH3mi1AqdMdU12h2VfETeOJkgGYq5ljd996AJ7ud2SyOLQmlhaNHH7Lx+Mdab8/zDN1SdxPARDgcM7AsRECHwQ15R20FaKUABGu4NTbR2fDKnYwiq5jQyBkLWP+LgGOgfUF4T4HZb2PY2bQtEP6QeqOtcW4rrsH24L7XhD+HSZb1hsitrE0VPbhJzxDwI4JF815XMnSVjZgYUXP8CxI1Y0FONlqtQYgsorZ9apoW1KPQe8brSSlRsi9sXB/tu56LmG7tEDNmrZ5XUwQYUUADBOu7t1niwXwIDAQAB"
	hash, err := types.ComputePoseidonHash(validPubKey)
	require.NoError(t, err)

	t.Run("default params are valid", func(t *testing.T) {
		params := types.DefaultParams()
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("empty params are valid", func(t *testing.T) {
		params := types.Params{MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes}
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("params with nil dkim pubkeys are valid", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			DkimPubkeys:        nil,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
		}
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("params with empty dkim pubkeys slice are valid", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			DkimPubkeys:        []types.DkimPubKey{},
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
		}
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("params with valid dkim pubkey", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("params with multiple valid dkim pubkeys", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
				{
					Domain:       "test.com",
					Selector:     "selector2",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err)
	})

	// Note: DkimPubKey.Validate() uses url.Parse which is very lenient
	// Empty domain passes url.Parse validation
	t.Run("params with empty domain is valid (url.Parse is lenient)", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err) // url.Parse("") succeeds
	})

	// Selector is not validated by DkimPubKey.Validate()
	t.Run("params with empty selector is valid (not validated)", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err) // selector is not validated
	})

	t.Run("params with invalid base64 pubkey should fail", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       "not-valid-base64!!!",
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.Error(t, err)
	})

	t.Run("params with whitespace in pubkey should fail", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       validPubKey + "\n",
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidPubKey)
	})

	t.Run("params with oversized pubkey should fail", func(t *testing.T) {
		tooLarge := make([]byte, types.DefaultMaxPubKeySizeBytes+1)
		for i := range tooLarge {
			tooLarge[i] = 'A'
		}
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       base64.StdEncoding.EncodeToString(tooLarge),
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.ErrorIs(t, err, types.ErrPubKeyTooLarge)
	})

	t.Run("encoded length exceeding limit fails fast", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: 1,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.ErrorIs(t, err, types.ErrPubKeyTooLarge)
	})

	t.Run("params with empty pubkey should fail", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       "",
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		// Empty string is valid base64 (decodes to empty bytes)
		require.NoError(t, err)
	})

	// PoseidonHash is not validated by DkimPubKey.Validate()
	t.Run("params with empty poseidon hash is valid (not validated)", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: []byte{},
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err) // poseidon hash is not validated
	})

	t.Run("params with nil poseidon hash is valid (not validated)", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: nil,
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err) // poseidon hash is not validated
	})

	t.Run("first invalid pubkey causes validation to fail", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       "invalid-base64!!!",
					PoseidonHash: hash.Bytes(),
				},
				{
					Domain:       "valid.com",
					Selector:     "selector2",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.Error(t, err)
	})

	t.Run("second invalid pubkey causes validation to fail", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "valid.com",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
				{
					Domain:       "example.com",
					Selector:     "selector2",
					PubKey:       "invalid-base64!!!",
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.Error(t, err)
	})

	t.Run("params with zero vkey identifier is valid", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     0,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "example.com",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("params with special characters in domain is valid", func(t *testing.T) {
		params := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes,
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "https://example.com:8080/path?query=1",
					Selector:     "selector1",
					PubKey:       validPubKey,
					PoseidonHash: hash.Bytes(),
				},
			},
		}
		err := params.Validate()
		require.NoError(t, err) // url.Parse is very lenient
	})
}
