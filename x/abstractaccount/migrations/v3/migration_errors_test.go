package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v3 "github.com/burnt-labs/xion/x/abstractaccount/migrations/v3"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func newMigrationStore(t *testing.T) (sdk.Context, *storetypes.KVStoreKey, codec.BinaryCodec) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey(types.TransientStoreKey))
	return testCtx.Ctx, key, codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

func TestMigrateStoreRequiresExistingParams(t *testing.T) {
	ctx, key, cdc := newMigrationStore(t)

	err := v3.MigrateStore(ctx, key, cdc)
	require.ErrorIs(t, err, types.ErrParsingParams)
	require.ErrorContains(t, err, "module params not found")
}

func TestMigrateStoreRejectsUndecodableParams(t *testing.T) {
	ctx, key, cdc := newMigrationStore(t)
	ctx.KVStore(key).Set(types.KeyParams, []byte{0xff, 0xff, 0xff})

	err := v3.MigrateStore(ctx, key, cdc)
	require.ErrorIs(t, err, types.ErrParsingParams)
	require.NotContains(t, err.Error(), "not found")
}

func TestMigrateStorePausesConfiguredRegistration(t *testing.T) {
	ctx, key, cdc := newMigrationStore(t)
	params, err := types.NewParamsWithAddressDerivationHash(false, []uint64{7}, 123, 456, make([]byte, 32))
	require.NoError(t, err)
	ctx.KVStore(key).Set(types.KeyParams, cdc.MustMarshal(params))

	require.NoError(t, v3.MigrateStore(ctx, key, cdc))

	var migrated types.Params
	require.NoError(t, cdc.Unmarshal(ctx.KVStore(key).Get(types.KeyParams), &migrated))
	require.Empty(t, migrated.AddressDerivationHash)
	require.False(t, migrated.RegistrationEnabled)
	require.Equal(t, []uint64{7}, migrated.AllowedCodeIDs)
	require.Equal(t, uint64(123), migrated.MaxGasBefore)
	require.Equal(t, uint64(456), migrated.MaxGasAfter)
}
