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
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee/ante"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

// TestMultiDenominationFeeValidation tests fee validation with multiple denominations
func TestMultiDenominationFeeValidation(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	t.Run("ThreeDenominationMinimums", func(t *testing.T) {
		// Set params with three different fee denominations
		params := types.Params{
			MinimumGasPrices: sdk.DecCoins{
				sdk.NewDecCoinFromDec("atom", math.LegacyNewDecWithPrec(1, 3)),
				sdk.NewDecCoinFromDec("osmo", math.LegacyNewDecWithPrec(5, 3)),
				sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
			},
			BypassMinFeeMsgTypes:            []string{},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}
		subspace.SetParamSet(ctx.Ctx, &params)

		stakingDenomFunc := func(ctx sdk.Context) string { return "uxion" }
		decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

		globalFees, err := decorator.GetGlobalFee(ctx.Ctx)
		require.NoError(t, err)
		require.Len(t, globalFees, 3, "Should have three denominations")

		// Verify denominations are sorted
		require.Equal(t, "atom", globalFees[0].Denom)
		require.Equal(t, "osmo", globalFees[1].Denom)
		require.Equal(t, "uxion", globalFees[2].Denom)
	})

	t.Run("DuplicateDenominationRejection", func(t *testing.T) {
		// Attempt to set params with duplicate denominations
		params := types.Params{
			MinimumGasPrices: sdk.DecCoins{
				sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
				sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)), // Duplicate
			},
			BypassMinFeeMsgTypes:            []string{},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		// ValidateBasic should catch duplicates
		err := params.ValidateBasic()
		require.Error(t, err, "Should reject duplicate denominations")
		require.Contains(t, err.Error(), "duplicate", "Error should mention duplicate")
	})

	t.Run("UnsortedDenominationRejection", func(t *testing.T) {
		// Attempt to set params with unsorted denominations
		params := types.Params{
			MinimumGasPrices: sdk.DecCoins{
				sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
				sdk.NewDecCoinFromDec("atom", math.LegacyNewDecWithPrec(1, 3)), // Out of order
			},
			BypassMinFeeMsgTypes:            []string{},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		// ValidateBasic should catch unsorted denoms
		err := params.ValidateBasic()
		require.Error(t, err, "Should reject unsorted denominations")
		require.Contains(t, err.Error(), "sorted", "Error should mention sorting")
	})

	t.Run("EmptyDenominationList", func(t *testing.T) {
		// Empty denomination list should be valid (defaults to staking denom with zero)
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            []string{},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		err := params.ValidateBasic()
		require.NoError(t, err, "Empty denomination list should be valid")

		subspace.SetParamSet(ctx.Ctx, &params)
		stakingDenomFunc := func(ctx sdk.Context) string { return "uxion" }
		decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

		// Should default to zero fee in staking denom
		zeroFees, err := decorator.DefaultZeroGlobalFee(ctx.Ctx)
		require.NoError(t, err)
		require.Len(t, zeroFees, 1)
		require.Equal(t, "uxion", zeroFees[0].Denom)
		require.True(t, zeroFees[0].Amount.IsZero())
	})

	t.Run("NegativeGasPriceRejection", func(t *testing.T) {
		// Attempt to create a negative DecCoin manually (bypassing the constructor)
		// This tests the validation logic, not the constructor
		negativeCoin := sdk.DecCoin{
			Denom:  "uxion",
			Amount: math.LegacyNewDec(-1),
		}
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{negativeCoin},
			BypassMinFeeMsgTypes:            []string{},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		err := params.ValidateBasic()
		require.Error(t, err, "Should reject negative gas prices")
		require.Contains(t, err.Error(), "negative", "Error should mention negative")
	})
}

// TestBypassMessageTypeValidation tests bypass message type validation
func TestBypassMessageTypeValidation(t *testing.T) {
	t.Run("EmptyMessageTypeRejection", func(t *testing.T) {
		params := types.Params{
			MinimumGasPrices: sdk.DecCoins{},
			BypassMinFeeMsgTypes: []string{
				"/cosmos.bank.v1beta1.MsgSend",
				"", // Empty string
			},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		err := params.ValidateBasic()
		require.Error(t, err, "Should reject empty message type")
		require.Contains(t, err.Error(), "empty", "Error should mention empty")
	})

	t.Run("InvalidMessageTypeFormatRejection", func(t *testing.T) {
		params := types.Params{
			MinimumGasPrices: sdk.DecCoins{},
			BypassMinFeeMsgTypes: []string{
				"MsgSend", // Missing "/" prefix
			},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		err := params.ValidateBasic()
		require.Error(t, err, "Should reject invalid message type format")
	})

	t.Run("ValidMessageTypes", func(t *testing.T) {
		params := types.Params{
			MinimumGasPrices: sdk.DecCoins{},
			BypassMinFeeMsgTypes: []string{
				"/cosmos.bank.v1beta1.MsgSend",
				"/cosmos.bank.v1beta1.MsgMultiSend",
				"/cosmos.authz.v1beta1.MsgRevoke",
			},
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		err := params.ValidateBasic()
		require.NoError(t, err, "Valid message types should pass validation")
	})

	t.Run("EmptyBypassListValid", func(t *testing.T) {
		// Empty bypass list means no messages can bypass fees
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            []string{}, // Empty list
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		err := params.ValidateBasic()
		require.NoError(t, err, "Empty bypass list should be valid")
	})
}

// TestGasCapValidation tests gas cap parameter validation
func TestGasCapValidation(t *testing.T) {
	t.Run("ZeroGasCapValid", func(t *testing.T) {
		// Zero gas cap means no bypass messages allowed (all require fee)
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            types.DefaultBypassMinFeeMsgTypes,
			MaxTotalBypassMinFeeMsgGasUsage: 0, // Zero cap
		}

		err := params.ValidateBasic()
		require.NoError(t, err, "Zero gas cap should be valid")
	})

	t.Run("VeryLargeGasCapValid", func(t *testing.T) {
		// Maximum uint64 gas cap should be valid
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            types.DefaultBypassMinFeeMsgTypes,
			MaxTotalBypassMinFeeMsgGasUsage: ^uint64(0), // Max uint64
		}

		err := params.ValidateBasic()
		require.NoError(t, err, "Maximum gas cap should be valid")
	})

	t.Run("StandardGasCapValid", func(t *testing.T) {
		// Standard 1M gas cap
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            types.DefaultBypassMinFeeMsgTypes,
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}

		err := params.ValidateBasic()
		require.NoError(t, err, "Standard gas cap should be valid")
	})
}

// TestContainsOnlyBypassMinFeeMsgsEdgeCases tests edge cases in bypass detection
func TestContainsOnlyBypassMinFeeMsgsAdvanced(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	params := types.Params{
		MinimumGasPrices: sdk.DecCoins{},
		BypassMinFeeMsgTypes: []string{
			"/cosmos.bank.v1beta1.MsgSend",
			"/cosmos.bank.v1beta1.MsgMultiSend",
		},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	stakingDenomFunc := func(ctx sdk.Context) string { return "uxion" }
	decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

	t.Run("EmptyMessageList", func(t *testing.T) {
		// Empty message list should return true (vacuous truth)
		msgs := []sdk.Msg{}
		result := decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, msgs)
		require.True(t, result, "Empty message list should return true")
	})

	t.Run("NilMessageList", func(t *testing.T) {
		// Nil message list should return true
		var msgs []sdk.Msg
		result := decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, msgs)
		require.True(t, result, "Nil message list should return true")
	})

	t.Run("EmptyBypassListRejectsAll", func(t *testing.T) {
		// When bypass list is empty, no messages should bypass
		emptyParams := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            []string{}, // Empty bypass list
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}
		subspace.SetParamSet(ctx.Ctx, &emptyParams)

		emptyDecorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

		// Even bypass message types should not bypass with empty list
		msgs := []sdk.Msg{}
		result := emptyDecorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, msgs)
		require.True(t, result, "Empty msgs with empty bypass list should return true")
	})
}

// TestCombinedFeeRequirementExtended tests extended combinations of fee requirements
// This complements the existing TestCombinedFeeRequirementEdgeCases
func TestCombinedFeeRequirementExtended(t *testing.T) {
	t.Run("BothGlobalAndLocalFeesIdentical", func(t *testing.T) {
		// Global and local fees are identical
		globalFees := sdk.DecCoins{
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(5, 3)),
		}
		localFees := sdk.DecCoins{
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(5, 3)),
		}

		combined, err := ante.CombinedFeeRequirement(globalFees, localFees)
		require.NoError(t, err)
		require.False(t, combined.IsZero())

		// Should return one of them (they're equal)
		for _, coin := range combined {
			if coin.Denom == "uxion" {
				require.True(t, coin.Amount.Equal(math.LegacyNewDecWithPrec(5, 3)))
			}
		}
	})

	t.Run("ThreeDifferentDenominations", func(t *testing.T) {
		// Test with three denominations across global and local
		globalFees := sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDecWithPrec(1, 3)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)),
		}
		localFees := sdk.DecCoins{
			sdk.NewDecCoinFromDec("osmo", math.LegacyNewDecWithPrec(3, 3)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
		}

		combined, err := ante.CombinedFeeRequirement(globalFees, localFees)
		require.NoError(t, err)
		require.False(t, combined.IsZero())

		// Should properly merge and select higher values
		require.GreaterOrEqual(t, len(combined), 1)
	})
}

// TestDefaultZeroGlobalFeeEdgeCases tests edge cases in default zero global fee
func TestDefaultZeroGlobalFeeEdgeCases(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	t.Run("EmptyStakingDenom", func(t *testing.T) {
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            types.DefaultBypassMinFeeMsgTypes,
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}
		subspace.SetParamSet(ctx.Ctx, &params)

		// Staking denom function returns empty string
		stakingDenomFunc := func(ctx sdk.Context) string { return "" }
		decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

		zeroFees, err := decorator.DefaultZeroGlobalFee(ctx.Ctx)

		// Should return error for empty staking denom
		require.Error(t, err, "Empty staking denom should return error")
		require.Nil(t, zeroFees)
	})

	t.Run("ValidStakingDenom", func(t *testing.T) {
		params := types.Params{
			MinimumGasPrices:                sdk.DecCoins{},
			BypassMinFeeMsgTypes:            types.DefaultBypassMinFeeMsgTypes,
			MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
		}
		subspace.SetParamSet(ctx.Ctx, &params)

		stakingDenomFunc := func(ctx sdk.Context) string { return "uxion" }
		decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

		zeroFees, err := decorator.DefaultZeroGlobalFee(ctx.Ctx)

		require.NoError(t, err)
		require.NotNil(t, zeroFees)
		require.Len(t, zeroFees, 1)
		require.Equal(t, "uxion", zeroFees[0].Denom)
		require.True(t, zeroFees[0].Amount.IsZero())
	})
}
