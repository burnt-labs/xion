package v2_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	v2 "github.com/burnt-labs/xion/x/dkim/migrations/v2"
	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestMigrateStore(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	tkey := storetypes.NewTransientStoreKey("transient_test")
	testCtx := testutil.DefaultContextWithDB(t, key, tkey)
	ctx := testCtx.Ctx

	storeService := runtime.NewKVStoreService(key)
	sb := collections.NewSchemaBuilder(storeService)

	paramsCollection := collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](encCfg.Codec))
	dkimPubKeys := collections.NewMap(
		sb,
		types.DkimPrefix,
		"dkim_pubkeys",
		collections.PairKeyCodec(collections.StringKey, collections.StringKey),
		codec.CollValue[types.DkimPubKey](encCfg.Codec),
	)

	_, err := sb.Build()
	require.NoError(t, err)

	t.Run("migrate params with default PublicInputIndices", func(t *testing.T) {
		// Set params without PublicInputIndices (simulating v1 state)
		oldParams := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: 2048,
			// PublicInputIndices is nil (v1 state)
		}
		err := paramsCollection.Set(ctx, oldParams)
		require.NoError(t, err)

		// Run migration
		err = v2.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection, dkimPubKeys)
		require.NoError(t, err)

		// Verify params have PublicInputIndices
		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, newParams.PublicInputIndices)
		require.Equal(t, types.DefaultPublicInputIndices().MinLength, newParams.PublicInputIndices.MinLength)
	})

	t.Run("migrate with no existing params sets defaults", func(t *testing.T) {
		// Clear params
		err := paramsCollection.Remove(ctx)
		require.NoError(t, err)

		// Run migration
		err = v2.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection, dkimPubKeys)
		require.NoError(t, err)

		// Verify default params are set
		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, newParams.PublicInputIndices)
	})

	t.Run("migrate clears PoseidonHash from DKIM records", func(t *testing.T) {
		// Set params
		err := paramsCollection.Set(ctx, types.DefaultParams())
		require.NoError(t, err)

		// Add DKIM records with PoseidonHash set
		testPubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
		hash, err := types.ComputePoseidonHash(testPubKey)
		require.NoError(t, err)

		dkimKey1 := types.DkimPubKey{
			Domain:       "test.com",
			Selector:     "selector1",
			PubKey:       testPubKey,
			PoseidonHash: hash.Bytes(), // v1 state: hash is stored
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		}
		dkimKey2 := types.DkimPubKey{
			Domain:       "test.com",
			Selector:     "selector2",
			PubKey:       testPubKey,
			PoseidonHash: hash.Bytes(), // v1 state: hash is stored
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		}

		key1 := collections.Join(dkimKey1.Domain, dkimKey1.Selector)
		key2 := collections.Join(dkimKey2.Domain, dkimKey2.Selector)
		//nolint:govet // copylocks: unavoidable in tests
		err = dkimPubKeys.Set(ctx, key1, dkimKey1)
		require.NoError(t, err)
		//nolint:govet // copylocks: unavoidable in tests
		err = dkimPubKeys.Set(ctx, key2, dkimKey2)
		require.NoError(t, err)

		// Verify records have PoseidonHash before migration
		record1, err := dkimPubKeys.Get(ctx, key1)
		require.NoError(t, err)
		require.NotEmpty(t, record1.PoseidonHash)

		// Run migration
		err = v2.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection, dkimPubKeys)
		require.NoError(t, err)

		// Verify PoseidonHash is cleared after migration
		record1After, err := dkimPubKeys.Get(ctx, key1)
		require.NoError(t, err)
		require.Empty(t, record1After.PoseidonHash)

		record2After, err := dkimPubKeys.Get(ctx, key2)
		require.NoError(t, err)
		require.Empty(t, record2After.PoseidonHash)

		// Verify other fields are preserved
		require.Equal(t, "test.com", record1After.Domain)
		require.Equal(t, "selector1", record1After.Selector)
		require.Equal(t, testPubKey, record1After.PubKey)
		require.Equal(t, types.Version_VERSION_DKIM1_UNSPECIFIED, record1After.Version)
		require.Equal(t, types.KeyType_KEY_TYPE_RSA_UNSPECIFIED, record1After.KeyType)
	})

	t.Run("migrate with records that have no PoseidonHash", func(t *testing.T) {
		// Set params
		err := paramsCollection.Set(ctx, types.DefaultParams())
		require.NoError(t, err)

		// Add DKIM record without PoseidonHash (edge case)
		testPubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

		dkimKey := types.DkimPubKey{
			Domain:       "nohash.com",
			Selector:     "selector",
			PubKey:       testPubKey,
			PoseidonHash: nil, // No hash stored
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		}

		key := collections.Join(dkimKey.Domain, dkimKey.Selector)
		//nolint:govet // copylocks: unavoidable in tests
		err = dkimPubKeys.Set(ctx, key, dkimKey)
		require.NoError(t, err)

		// Run migration
		err = v2.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection, dkimPubKeys)
		require.NoError(t, err)

		// Verify record is unchanged (still no hash)
		recordAfter, err := dkimPubKeys.Get(ctx, key)
		require.NoError(t, err)
		require.Empty(t, recordAfter.PoseidonHash)
		require.Equal(t, "nohash.com", recordAfter.Domain)
		require.Equal(t, testPubKey, recordAfter.PubKey)
	})
}
