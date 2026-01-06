package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestGenesis(t *testing.T) {
	f := SetupTest(t)
	hash, err := types.ComputePoseidonHash(testPubKey)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		request *types.GenesisState
		err     bool
	}{
		{
			name:    "success, only default params",
			request: &types.GenesisState{},
			err:     false,
		},
		{
			name: "success, with revoked pubkey",
			request: &types.GenesisState{
				RevokedPubkeys: []string{testPubKey},
				Params:         types.Params{},
			},
			err: false,
		},
		{
			name: "success, with dkim records",
			request: &types.GenesisState{
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "x.com",
						Selector:     "test",
						PubKey:       testPubKey,
						PoseidonHash: []byte(hash.String()),
					},
				},
				Params: types.Params{},
			},
			err: false,
		},
		{
			name: "fail, invalid dkim record",
			request: &types.GenesisState{
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "x.com",
						Selector:     "test",
						PubKey:       "invalid",
						PoseidonHash: hash.Bytes(),
					},
				},
				Params: types.Params{},
			},
			err: true,
		},
		{
			name: "fail, invalid data",
			request: &types.GenesisState{
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "x.com",
						PubKey:       "test!@#", // invalid base64 characters
						Selector:     "test",
						PoseidonHash: []byte("test"),
					},
				},
				Params: types.Params{},
			},
			err: true,
		},
		{
			name: "fail, invalid revoked pubkey",
			request: &types.GenesisState{
				RevokedPubkeys: []string{"not-base64"},
				Params:         types.Params{},
			},
			err: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err {
				require.Error(t, f.k.InitGenesis(f.ctx, tc.request))
			} else {
				require.NoError(t, f.k.InitGenesis(f.ctx, tc.request))
			}
		})
	}

	got := f.k.ExportGenesis(f.ctx)
	require.NotNil(t, got)
}

// Helper function to find a DKIM record by domain and selector
func findDkimRecord(records []types.DkimPubKey, domain, selector string) *types.DkimPubKey {
	for _, pk := range records {
		if pk.Domain == domain && pk.Selector == selector {
			return &pk
		}
	}
	return nil
}

// Helper function to count DKIM records by domain
func countRecordsByDomain(records []types.DkimPubKey, domain string) int {
	count := 0
	for _, pk := range records {
		if pk.Domain == domain {
			count++
		}
	}
	return count
}

func TestInitGenesisExtended(t *testing.T) {
	hash, err := types.ComputePoseidonHash(testPubKey)
	require.NoError(t, err)

	t.Run("init with multiple dkim records", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "init-test1.com",
					Selector:     "selector1",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
				},
				{
					Domain:       "init-test2.com",
					Selector:     "selector2",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
				},
				{
					Domain:       "init-test3.com",
					Selector:     "selector3",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		// Verify all records were stored by finding them
		exported := f.k.ExportGenesis(f.ctx)
		require.NotNil(t, findDkimRecord(exported.DkimPubkeys, "init-test1.com", "selector1"))
		require.NotNil(t, findDkimRecord(exported.DkimPubkeys, "init-test2.com", "selector2"))
		require.NotNil(t, findDkimRecord(exported.DkimPubkeys, "init-test3.com", "selector3"))
	})

	t.Run("init with same domain different selectors", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "multi-selector-test.com",
					Selector:     "selector1",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
				{
					Domain:       "multi-selector-test.com",
					Selector:     "selector2",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		// Verify both records were stored (different selectors)
		exported := f.k.ExportGenesis(f.ctx)
		count := countRecordsByDomain(exported.DkimPubkeys, "multi-selector-test.com")
		require.Equal(t, 2, count)
	})

	t.Run("init with revoked pubkeys stored", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			RevokedPubkeys: []string{testPubKey},
			Params:         types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		has, err := f.k.RevokedKeys.Has(f.ctx, testPubKey)
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("init with nil genesis state panics", func(t *testing.T) {
		f := SetupTest(t)

		// This should panic
		require.Panics(t, func() {
			_ = f.k.InitGenesis(f.ctx, nil)
		})
	})

	t.Run("init preserves version and key type", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "version-test.com",
					Selector:     "selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		exported := f.k.ExportGenesis(f.ctx)
		record := findDkimRecord(exported.DkimPubkeys, "version-test.com", "selector")
		require.NotNil(t, record)
		require.Equal(t, types.Version_VERSION_DKIM1_UNSPECIFIED, record.Version)
		require.Equal(t, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, record.KeyType)
	})

	t.Run("init overwrites existing records with same key", func(t *testing.T) {
		f := SetupTest(t)

		// First init
		genesis1 := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "overwrite-test.com",
					Selector:     "selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte("hash1"),
				},
			},
			Params: types.Params{},
		}
		err := f.k.InitGenesis(f.ctx, genesis1)
		require.NoError(t, err)

		// Second init with same domain/selector but different hash
		genesis2 := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "overwrite-test.com",
					Selector:     "selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte("hash2"),
				},
			},
			Params: types.Params{},
		}
		err = f.k.InitGenesis(f.ctx, genesis2)
		require.NoError(t, err)

		// Verify record was overwritten
		exported := f.k.ExportGenesis(f.ctx)
		record := findDkimRecord(exported.DkimPubkeys, "overwrite-test.com", "selector")
		require.NotNil(t, record)
		require.Equal(t, []byte("hash2"), record.PoseidonHash)

		// Count should be 1 for this domain
		count := countRecordsByDomain(exported.DkimPubkeys, "overwrite-test.com")
		require.Equal(t, 1, count)
	})
}

func TestExportGenesisExtended(t *testing.T) {
	hash, err := types.ComputePoseidonHash(testPubKey)
	require.NoError(t, err)

	t.Run("export genesis returns non-nil", func(t *testing.T) {
		f := SetupTest(t)

		exported := f.k.ExportGenesis(f.ctx)
		require.NotNil(t, exported)
		require.NotNil(t, exported.Params)
	})

	t.Run("export genesis with added record", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			RevokedPubkeys: []string{testPubKey},
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "export-test.com",
					Selector:     "selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		exported := f.k.ExportGenesis(f.ctx)
		require.NotNil(t, exported)

		// Find the record we added
		record := findDkimRecord(exported.DkimPubkeys, "export-test.com", "selector")
		require.NotNil(t, record)
		require.Equal(t, testPubKey, record.PubKey)
		require.Contains(t, exported.RevokedPubkeys, testPubKey)
	})

	t.Run("export genesis with multiple records", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "export-multi1.com",
					Selector:     "selector1",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
				{
					Domain:       "export-multi2.com",
					Selector:     "selector2",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
				{
					Domain:       "export-multi3.com",
					Selector:     "selector3",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		exported := f.k.ExportGenesis(f.ctx)
		require.NotNil(t, exported)

		// Verify all domains are present
		require.NotNil(t, findDkimRecord(exported.DkimPubkeys, "export-multi1.com", "selector1"))
		require.NotNil(t, findDkimRecord(exported.DkimPubkeys, "export-multi2.com", "selector2"))
		require.NotNil(t, findDkimRecord(exported.DkimPubkeys, "export-multi3.com", "selector3"))
	})

	t.Run("export preserves all fields", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "preserve-fields.com",
					Selector:     "selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		exported := f.k.ExportGenesis(f.ctx)
		require.NotNil(t, exported)

		pk := findDkimRecord(exported.DkimPubkeys, "preserve-fields.com", "selector")
		require.NotNil(t, pk)
		require.Equal(t, "preserve-fields.com", pk.Domain)
		require.Equal(t, "selector", pk.Selector)
		require.Equal(t, testPubKey, pk.PubKey)
		require.Equal(t, []byte(hash.String()), pk.PoseidonHash)
		require.Equal(t, types.Version_VERSION_DKIM1_UNSPECIFIED, pk.Version)
		require.Equal(t, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, pk.KeyType)
	})
}

// ============================================================================
// Genesis Validation Edge Cases
// ============================================================================

func TestGenesisValidation(t *testing.T) {
	hash, err := types.ComputePoseidonHash(testPubKey)
	require.NoError(t, err)

	t.Run("valid genesis with all fields", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "validation-test.com",
					Selector:     "dkim1",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
					Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
					KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)
	})

	t.Run("invalid pubkey format fails", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "invalid-pk.com",
					Selector:     "selector",
					PubKey:       "not-valid-base64!@#$%",
					PoseidonHash: []byte(hash.String()),
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.Error(t, err)
	})

	t.Run("multiple records with one invalid fails", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "valid.com",
					Selector:     "selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
				{
					Domain:       "invalid.com",
					Selector:     "selector",
					PubKey:       "invalid!@#",
					PoseidonHash: []byte(hash.String()),
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.Error(t, err)
	})

	t.Run("special characters in selector allowed", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "special-selector.com",
					Selector:     "dkim-2023_selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		// Verify it was stored
		exported := f.k.ExportGenesis(f.ctx)
		record := findDkimRecord(exported.DkimPubkeys, "special-selector.com", "dkim-2023_selector")
		require.NotNil(t, record)
	})

	t.Run("subdomain is allowed", func(t *testing.T) {
		f := SetupTest(t)

		genesis := &types.GenesisState{
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:       "mail.subdomain.example.com",
					Selector:     "selector",
					PubKey:       testPubKey,
					PoseidonHash: []byte(hash.String()),
				},
			},
			Params: types.Params{},
		}

		err := f.k.InitGenesis(f.ctx, genesis)
		require.NoError(t, err)

		// Verify it was stored
		exported := f.k.ExportGenesis(f.ctx)
		record := findDkimRecord(exported.DkimPubkeys, "mail.subdomain.example.com", "selector")
		require.NotNil(t, record)
	})
}
