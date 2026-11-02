package app

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/appmodule"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func feegrantEndBlocker(t *testing.T, gapp *WasmApp) appmodule.HasEndBlocker {
	t.Helper()
	endBlocker, ok := gapp.ModuleManager.Modules[feegrant.ModuleName].(appmodule.HasEndBlocker)
	require.True(t, ok)
	return endBlocker
}

// TestLegacyFeeAllowanceQueueFixWaitsForExpiry: before the first legacy entry
// expires the check writes nothing; from its expiry on (inclusive, as the SDK
// range is) it rewrites the legacy entries and the feegrant EndBlocker applies.
func TestLegacyFeeAllowanceQueueFixWaitsForExpiry(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false).
		WithChainID("xion-mainnet-1").
		WithBlockHeight(24_800_000).
		WithBlockTime(time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC))
	expired, later := setupFeeAllowanceQueue(t, gapp, ctx)
	endBlocker := feegrantEndBlocker(t, gapp)

	before, _ := ctx.WithBlockTime(expired.Add(-time.Nanosecond)).CacheContext()
	require.NoError(t, gapp.fixDueLegacyFeeAllowanceQueue(before))
	requireLegacyQueueValues(t, gapp, before, []byte{})
	require.NoError(t, endBlocker.EndBlock(before))

	control, _ := ctx.WithBlockTime(expired).CacheContext()
	require.ErrorContains(t, endBlocker.EndBlock(control), "wanted size to be at least 1")

	at := ctx.WithBlockTime(expired)
	require.NoError(t, gapp.fixDueLegacyFeeAllowanceQueue(at))
	requireLegacyQueueValues(t, gapp, at, []byte{1})
	require.NoError(t, endBlocker.EndBlock(at))
	for _, grantee := range []sdk.AccAddress{queueLegacyGrantee, queueCurrentGrantee} {
		_, err := gapp.FeeGrantKeeper.GetAllowance(at, queueGranter, grantee)
		require.ErrorIs(t, err, collections.ErrNotFound, grantee.String())
	}
	_, err := gapp.FeeGrantKeeper.GetAllowance(at, queueGranter, queueLaterGrantee)
	require.NoError(t, err)

	after := at.WithBlockTime(later)
	require.NoError(t, gapp.fixDueLegacyFeeAllowanceQueue(after))
	require.NoError(t, endBlocker.EndBlock(after))
	_, err = gapp.FeeGrantKeeper.GetAllowance(after, queueGranter, queueLaterGrantee)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

// TestLegacyFeeAllowanceQueueFixFollowsPruneLimit: with 200 expired entries
// ahead of a legacy one, the feegrant EndBlocker stops before decoding it, so
// the check must not fire in that block; it fires in the next one.
func TestLegacyFeeAllowanceQueueFixFollowsPruneLimit(t *testing.T) {
	gapp := Setup(t)
	start := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	ctx := gapp.NewContext(false).WithChainID("xion-mainnet-1").WithBlockTime(start)
	granter := sdk.AccAddress([]byte("limit_granter_______"))
	early := start.Add(time.Minute)
	for i := 0; i < feegrantEndBlockPruneLimit; i++ {
		grantee := sdk.AccAddress([]byte{0xA0, byte(i >> 8), byte(i), 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
		gapp.AccountKeeper.SetAccount(ctx, gapp.AccountKeeper.NewAccountWithAddress(ctx, grantee))
		require.NoError(t, gapp.FeeGrantKeeper.GrantAllowance(ctx, granter, grantee, &feegrant.BasicAllowance{Expiration: &early}))
	}
	legacyExp := start.Add(2 * time.Minute)
	gapp.AccountKeeper.SetAccount(ctx, gapp.AccountKeeper.NewAccountWithAddress(ctx, queueLegacyGrantee))
	require.NoError(t, gapp.FeeGrantKeeper.GrantAllowance(ctx, granter, queueLegacyGrantee, &feegrant.BasicAllowance{Expiration: &legacyExp}))
	legacyKey := collections.Join3(legacyExp, queueLegacyGrantee, granter)
	rawKey, err := collections.EncodeKeyWithPrefix(feegrant.FeeAllowanceQueueKeyPrefix, gapp.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec(), legacyKey)
	require.NoError(t, err)
	store := ctx.KVStore(gapp.GetKey(feegrant.StoreKey))
	store.Set(rawKey, []byte{})

	endBlocker := feegrantEndBlocker(t, gapp)
	first := ctx.WithBlockTime(legacyExp)
	require.NoError(t, gapp.fixDueLegacyFeeAllowanceQueue(first))
	require.Equal(t, []byte{}, store.Get(rawKey))
	require.NoError(t, endBlocker.EndBlock(first))

	second := first.WithBlockTime(legacyExp.Add(4 * time.Second))
	require.NoError(t, gapp.fixDueLegacyFeeAllowanceQueue(second))
	require.Equal(t, []byte{1}, store.Get(rawKey))
	require.NoError(t, endBlocker.EndBlock(second))
	require.False(t, store.Has(rawKey))
}

// xionMainnetLegacyQueueKeys are the FeeAllowanceQueue keys xion-mainnet-1
// held with empty values at height 24753295 (2026-10-30), read from the
// archive node's /store/feegrant/subspace prefix 0x01.
var xionMainnetLegacyQueueKeys = []string{
	"01323032362d31312d30345431303a34393a34382e303030303030303030140749b376b650b501951570425a1d912a0213ed6b2003c920914294c001d43cfb3e190471efb07d8544311ada6b01c12614d6090fe5",
	"01323032362d31312d31355431343a30373a34362e303030303030303030149a7ff129d2c021c6421e77a90d49e2acc4c99d3220919c2354067161aa410e93fd1696d46ee70d0eb80bbf7328b9740da9ef2c067b",
	"01323032362d31312d31375431313a35353a31372e30303030303030303014a4749beef30e17f2d1a797fcc9c828c401f3f97e202367d478420f1c8276e13b11cee14adefccd8560b390b5190c9f2be66fcb7731",
	"01323032362d31322d30325430373a33353a35382e30303030303030303014ef95f4e3917fed54e8b3956fce5ab89734a470b820ab0f43c53e5584895645cf9f952a1447981f4e8cba0a66fb019d4be0c825f72a",
	"01323032362d31322d31355431323a33373a33382e303030303030303030147b42728e78b043377d0492d083012a00f270eb3420ab0f43c53e5584895645cf9f952a1447981f4e8cba0a66fb019d4be0c825f72a",
	"01323032362d31322d33305431303a33313a35342e30303030303030303014066b4f02803b9b0daa2a9a57d84a6034f7243cbe2091c5b8f9261f5dda02557913566aeaa6702b2210a6f452cf875c5a55a5b2990d",
	"01323032372d30312d30355431373a31363a30302e30303030303030303014b90d88f5ec1a18ce892f2172bb75040808f8c52d201f65e9c4a761dd3721492c52527d097b0356c561f508a2c04357baed41a01cee",
	"01323032372d30312d31325430363a32353a33342e30303030303030303014ad068661c4286fe7781e7ff35f10cbb34c51b9cf20ae3350900e3173769cb2d541bb6fd85e678c6912654d7cd378ca258ef298b9c5",
	"01323032372d30312d31345431343a32323a35322e30303030303030303014ae290d5ffcf13f41ee51f40021b5793b847699b0200f6fdd5c715d2fd453e43947c02b63e5519143573fc9be4c20bc8a3b375e1932",
}

// TestLegacyFeeAllowanceQueueFixOnMainnetEntries grants the nine mainnet
// allowances, checks the keeper writes exactly the mainnet keys, empties their
// values the way the pre-v32 module stored them, and runs the blocks around
// the first expiry, 2026-11-04T10:49:48Z.
func TestLegacyFeeAllowanceQueueFixOnMainnetEntries(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false).
		WithChainID("xion-mainnet-1").
		WithBlockTime(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	keyCodec := gapp.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec()
	store := ctx.KVStore(gapp.GetKey(feegrant.StoreKey))
	var raw [][]byte
	for _, h := range xionMainnetLegacyQueueKeys {
		k, err := hex.DecodeString(h)
		require.NoError(t, err)
		_, key, err := keyCodec.Decode(k[len(feegrant.FeeAllowanceQueueKeyPrefix):])
		require.NoError(t, err)
		exp, grantee, granter := key.K1(), key.K2(), key.K3()
		gapp.AccountKeeper.SetAccount(ctx, gapp.AccountKeeper.NewAccountWithAddress(ctx, grantee))
		require.NoError(t, gapp.FeeGrantKeeper.GrantAllowance(ctx, granter, grantee, &feegrant.BasicAllowance{Expiration: &exp}))
		require.Equal(t, []byte{1}, store.Get(k), "keeper wrote a different key for %s", h)
		store.Set(k, []byte{})
		raw = append(raw, k)
	}
	it := storetypes.KVStorePrefixIterator(store, feegrant.FeeAllowanceQueueKeyPrefix)
	n := 0
	for ; it.Valid(); it.Next() {
		n++
	}
	require.NoError(t, it.Close())
	require.Equal(t, 9, n)

	endBlocker := feegrantEndBlocker(t, gapp)
	first := time.Date(2026, 11, 4, 10, 49, 48, 0, time.UTC)

	before := ctx.WithBlockTime(first.Add(-time.Millisecond))
	require.NoError(t, gapp.fixDueLegacyFeeAllowanceQueue(before))
	require.NoError(t, endBlocker.EndBlock(before))
	for _, k := range raw {
		require.Equal(t, []byte{}, store.Get(k))
	}

	at := ctx.WithBlockTime(first.Add(1500 * time.Millisecond))
	require.NoError(t, gapp.fixDueLegacyFeeAllowanceQueue(at))
	require.NoError(t, endBlocker.EndBlock(at))
	require.False(t, store.Has(raw[0]))
	for _, k := range raw[1:] {
		require.Equal(t, []byte{1}, store.Get(k))
	}
}

// TestEndBlockerRunsLegacyFeeAllowanceQueueFix: the app EndBlocker applies the
// block in which a legacy entry expires.
func TestEndBlockerRunsLegacyFeeAllowanceQueueFix(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false).
		WithChainID("xion-mainnet-1").
		WithBlockTime(time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC))
	expired, _ := setupFeeAllowanceQueue(t, gapp, ctx)
	at := ctx.WithBlockTime(expired)
	_, err := gapp.EndBlocker(at)
	require.NoError(t, err)
	_, err = gapp.FeeGrantKeeper.GetAllowance(at, queueGranter, queueLegacyGrantee)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
