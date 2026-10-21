package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee/ante"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

// newBypassDecorator returns a decorator whose only bypass message type is
// MsgSend, with the given global fee params stored (nil leaves the params unset).
func newBypassDecorator(t *testing.T, params *types.Params) (ante.FeeDecorator, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	testCtx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())
	if params != nil {
		subspace.SetParamSet(testCtx.Ctx, params)
	}

	decorator := ante.NewFeeDecorator(subspace, func(sdk.Context) string { return testStakingDenom })
	return decorator, testCtx.Ctx
}

func bypassParams(minGasPrices sdk.DecCoins, maxGas uint64) *types.Params {
	return &types.Params{
		MinimumGasPrices:                minGasPrices,
		BypassMinFeeMsgTypes:            []string{sdk.MsgTypeURL(&banktypes.MsgSend{})},
		MaxTotalBypassMinFeeMsgGasUsage: maxGas,
	}
}

func TestAnteHandleBypassMessages(t *testing.T) {
	bypassTx := func(gas uint64, fees sdk.Coins) mockFeeTx {
		return mockFeeTx{gas: gas, fees: fees, msgs: []sdk.Msg{&banktypes.MsgSend{}}}
	}
	globalFee := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}

	t.Run("gas above the bypass cap is rejected", func(t *testing.T) {
		decorator, ctx := newBypassDecorator(t, bypassParams(globalFee, 1_000))

		_, err := decorator.AnteHandle(ctx, bypassTx(1_001, nil), false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			t.Fatal("next handler must not run")
			return ctx, nil
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "bypass messages cannot use more than 1000 gas, but got 1001")
	})

	t.Run("fee in a denom outside the global fee set is rejected", func(t *testing.T) {
		decorator, ctx := newBypassDecorator(t, bypassParams(globalFee, 1_000_000))

		_, err := decorator.AnteHandle(ctx, bypassTx(100, sdk.NewCoins(sdk.NewInt64Coin("atom", 1))), false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			t.Fatal("next handler must not run")
			return ctx, nil
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "fee denom atom is not allowed for bypass messages")
	})

	t.Run("fee in an allowed denom passes through", func(t *testing.T) {
		decorator, ctx := newBypassDecorator(t, bypassParams(globalFee, 1_000_000))

		called := false
		_, err := decorator.AnteHandle(ctx, bypassTx(100, sdk.NewCoins(sdk.NewInt64Coin("uxion", 1))), false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			called = true
			return ctx, nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("zero-fee bypass clears the local minimum gas price", func(t *testing.T) {
		decorator, ctx := newBypassDecorator(t, bypassParams(globalFee, 1_000_000))
		ctx = ctx.WithMinGasPrices(sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(5, 2))})

		var seen sdk.DecCoins
		_, err := decorator.AnteHandle(ctx, bypassTx(100, nil), false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			seen = ctx.MinGasPrices()
			return ctx, nil
		})
		require.NoError(t, err)
		require.True(t, seen.IsZero())
	})

	t.Run("zero-fee bypass keeps an already empty minimum gas price", func(t *testing.T) {
		decorator, ctx := newBypassDecorator(t, bypassParams(globalFee, 1_000_000))

		called := false
		_, err := decorator.AnteHandle(ctx, bypassTx(100, nil), false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			called = true
			return ctx, nil
		})
		require.NoError(t, err)
		require.True(t, called)
	})
}

func TestGetMaxTotalBypassMinFeeMsgGasUsageDefault(t *testing.T) {
	decorator, ctx := newBypassDecorator(t, nil)
	require.Equal(t, types.DefaultmaxTotalBypassMinFeeMsgGasUsage, decorator.GetMaxTotalBypassMinFeeMsgGasUsage(ctx))

	decorator, ctx = newBypassDecorator(t, bypassParams(nil, 42))
	require.Equal(t, uint64(42), decorator.GetMaxTotalBypassMinFeeMsgGasUsage(ctx))
}
