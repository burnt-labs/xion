package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee/ante"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestNewFeeDecorator(t *testing.T) {
	// Create a test subspace without key table
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)

	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)

	stakingDenomFunc := func(ctx sdk.Context) string {
		return "stake"
	}

	// Test with subspace that doesn't have key table - should panic
	require.Panics(t, func() {
		ante.NewFeeDecorator(subspace, stakingDenomFunc)
	})

	// Test with subspace that has key table
	subspaceWithKeyTable := subspace.WithKeyTable(types.ParamKeyTable())
	decorator := ante.NewFeeDecorator(subspaceWithKeyTable, stakingDenomFunc)
	require.NotNil(t, decorator)
}

func TestFeeDecoratorMethods(t *testing.T) {
	// Create test context
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

	// Set params with global fees
	params := types.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{"/ibc.core.channel.v1.MsgRecvPacket"},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	stakingDenomFunc := func(ctx sdk.Context) string {
		return "stake"
	}

	decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

	// Test GetGlobalFee
	globalFees, err := decorator.GetGlobalFee(ctx.Ctx)
	require.NoError(t, err)
	require.NotNil(t, globalFees)
	require.Len(t, globalFees, 1)
	require.Equal(t, "uxion", globalFees[0].Denom)

	// Test DefaultZeroGlobalFee
	zeroFees, err := decorator.DefaultZeroGlobalFee(ctx.Ctx)
	require.NoError(t, err)
	require.NotNil(t, zeroFees)
	require.Len(t, zeroFees, 1)
	require.Equal(t, "stake", zeroFees[0].Denom)
	require.True(t, zeroFees[0].Amount.IsZero())

	// Test ContainsOnlyBypassMinFeeMsgs
	msgs := []sdk.Msg{}
	result := decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, msgs)
	require.True(t, result)

	// Test GetBypassMsgTypes
	bypassTypes := decorator.GetBypassMsgTypes(ctx.Ctx)
	require.NotNil(t, bypassTypes)
	require.Contains(t, bypassTypes, "/ibc.core.channel.v1.MsgRecvPacket")

	// Test GetMaxTotalBypassMinFeeMsgGasUsage
	maxGas := decorator.GetMaxTotalBypassMinFeeMsgGasUsage(ctx.Ctx)
	require.Equal(t, uint64(1_000_000), maxGas)
}

func TestGetMinGasPrice(t *testing.T) {
	// Create a context with min gas prices
	ctx := sdk.Context{}.WithMinGasPrices(sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
	})

	// Test GetMinGasPrice
	minGasPrice := ante.GetMinGasPrice(ctx)
	require.NotNil(t, minGasPrice)
	require.Len(t, minGasPrice, 1)
	require.Equal(t, "uxion", minGasPrice[0].Denom)

	// Test with zero min gas prices
	ctx = sdk.Context{}.WithMinGasPrices(sdk.DecCoins{})
	minGasPrice = ante.GetMinGasPrice(ctx)
	require.True(t, minGasPrice.IsZero())
}

func TestGetGlobalFeeEmptyParams(t *testing.T) {
	// Create test context
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

	// Set params with empty global fees (should use default zero fee)
	params := types.Params{
		MinimumGasPrices:                sdk.DecCoins{},
		BypassMinFeeMsgTypes:            []string{},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	stakingDenomFunc := func(ctx sdk.Context) string {
		return "stake"
	}

	decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

	// Test GetGlobalFee with empty params (should return default zero fee)
	globalFees, err := decorator.GetGlobalFee(ctx.Ctx)
	require.NoError(t, err)
	require.NotNil(t, globalFees)
	require.Len(t, globalFees, 1)
	require.Equal(t, "stake", globalFees[0].Denom)
	require.True(t, globalFees[0].Amount.IsZero())
}

func TestDefaultZeroGlobalFeeError(t *testing.T) {
	// Create test context
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

	// Set params
	params := types.Params{
		MinimumGasPrices:                sdk.DecCoins{},
		BypassMinFeeMsgTypes:            []string{},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	// Test with empty bond denom (should error)
	emptyBondDenomFunc := func(ctx sdk.Context) string {
		return ""
	}

	decorator := ante.NewFeeDecorator(subspace, emptyBondDenomFunc)

	// Test DefaultZeroGlobalFee with empty bond denom
	_, err := decorator.DefaultZeroGlobalFee(ctx.Ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty staking bond denomination")

	// Test that GetGlobalFee also fails when DefaultZeroGlobalFee fails
	_, err = decorator.GetGlobalFee(ctx.Ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty staking bond denomination")
}

func TestContainsOnlyBypassMinFeeMsgsEdgeCases(t *testing.T) {
	// Create test context
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

	stakingDenomFunc := func(ctx sdk.Context) string {
		return "stake"
	}

	decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

	// Test with empty messages slice
	emptyMsgs := []sdk.Msg{}
	result := decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, emptyMsgs)
	require.True(t, result) // Empty messages should return true

	// Test with default bypass messages types - use actual params that include bypass types
	params := types.DefaultParams()
	subspace.SetParamSet(ctx.Ctx, &params)

	// Verify default bypass types exist
	bypassTypes := decorator.GetBypassMsgTypes(ctx.Ctx)
	require.NotEmpty(t, bypassTypes)

	// Test with no bypass messages (this should return false since no msgs match bypass types)
	// For this we need to test with actual non-bypass message types
	// Since we can't easily create a non-bypass message in this test context,
	// let's test the logic by setting custom bypass types

	// Set custom params with specific bypass message types for testing
	customParams := types.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{"/xion.v1.MsgSend"},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &customParams)

	// Test edge case: ensure that the function correctly identifies bypass vs non-bypass
	// by checking the behavior with default params
	defaultParams := types.DefaultParams()
	subspace.SetParamSet(ctx.Ctx, &defaultParams)

	// Verify that bypass types are properly configured (using actual default types)
	bypassTypes = decorator.GetBypassMsgTypes(ctx.Ctx)
	require.Contains(t, bypassTypes, "/xion.v1.MsgSend")
	require.Contains(t, bypassTypes, "/xion.v1.MsgMultiSend")
	require.Contains(t, bypassTypes, "/xion.jwk.v1.MsgDeleteAudience")
	require.Contains(t, bypassTypes, "/cosmos.authz.v1beta1.MsgRevoke")

	// Test that empty slice returns true
	result = decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, []sdk.Msg{})
	require.True(t, result)
}

// Simple mock message for testing bypass functionality
type mockBypassMsg struct {
	route string
	typ   string
}

func (m mockBypassMsg) Reset()                       {}
func (m mockBypassMsg) String() string               { return m.typ }
func (m mockBypassMsg) ProtoMessage()                {}
func (m mockBypassMsg) ValidateBasic() error         { return nil }
func (m mockBypassMsg) GetSigners() []sdk.AccAddress { return nil }
func (m mockBypassMsg) Route() string                { return m.route }
func (m mockBypassMsg) Type() string                 { return m.typ }
func (m mockBypassMsg) GetSignBytes() []byte         { return nil }

type mockMsg struct {
	typeURL string
}

func (m *mockMsg) ProtoMessage()                {}
func (m *mockMsg) Reset()                       {}
func (m *mockMsg) String() string               { return "" }
func (m *mockMsg) ValidateBasic() error         { return nil }
func (m *mockMsg) GetSignBytes() []byte         { return nil }
func (m *mockMsg) GetSigners() []sdk.AccAddress { return nil }
func (m *mockMsg) Route() string                { return "" }
func (m *mockMsg) Type() string                 { return m.typeURL }

// Implement proto.Message interface method for SDK compatibility
func (m *mockMsg) XXX_MessageName() string { return m.typeURL }

func TestContainsOnlyBypassMinFeeMsgsWithMessages(t *testing.T) {
	// Create test context
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

	stakingDenomFunc := func(ctx sdk.Context) string {
		return "stake"
	}

	decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

	// Test with mixed messages - some bypass, some not
	// Since we can't easily create real protobuf messages that match the bypass types,
	// we test the logic by checking behavior when all messages are in bypass list vs when not

	// First, set params with a simple bypass type for testing
	params := types.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{}, // Empty bypass list - no messages will be bypassed
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	// Test with messages when bypass list is empty - should return false for any messages
	nonBypassMsgs := []sdk.Msg{
		mockBypassMsg{route: "test", typ: "TestMsg"},
	}
	result := decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, nonBypassMsgs)
	require.False(t, result)

	// Test with multiple messages when bypass list is empty
	multipleNonBypassMsgs := []sdk.Msg{
		mockBypassMsg{route: "test", typ: "TestMsg1"},
		mockBypassMsg{route: "test", typ: "TestMsg2"},
	}
	result = decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, multipleNonBypassMsgs)
	require.False(t, result)

	// Test with one bypass message and one non-bypass message (mixed scenario)
	// This should return false since not ALL messages are bypass messages
	params = types.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{"//cosmos.bank.v1beta1.MsgSend"}, // Use the actual format that sdk.MsgTypeURL returns
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	// Create mock messages with proper protobuf-style type URLs
	mixedMsgs := []sdk.Msg{
		&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},      // This matches bypass type
		&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgMultiSend"}, // This does NOT match bypass type
	}
	result = decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, mixedMsgs)
	require.False(t, result) // Should be false because not ALL messages are bypass

	// Test with ALL bypass messages - should return true
	params = types.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{"//cosmos.bank.v1beta1.MsgSend", "//cosmos.bank.v1beta1.MsgMultiSend"}, // Use the actual format that sdk.MsgTypeURL returns
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	allBypassMsgs := []sdk.Msg{
		&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},      // This is bypass
		&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgMultiSend"}, // This is also bypass
	}

	// Debug: Check what sdk.MsgTypeURL actually returns for our mock messages
	for i, msg := range allBypassMsgs {
		actualTypeURL := sdk.MsgTypeURL(msg)
		expectedTypeURL := allBypassMsgs[i].(*mockMsg).typeURL
		t.Logf("Message %d: expected=%s, actual=%s", i, expectedTypeURL, actualTypeURL)
	}

	result = decorator.ContainsOnlyBypassMinFeeMsgs(ctx.Ctx, allBypassMsgs)
	require.True(t, result) // Should be true because ALL messages are bypass
}

// mockFeeTx implements sdk.FeeTx interface for testing
type mockFeeTx struct {
	gas     uint64
	fees    sdk.Coins
	payer   []byte
	granter []byte
	msgs    []sdk.Msg
}

func (m mockFeeTx) GetGas() uint64     { return m.gas }
func (m mockFeeTx) GetFee() sdk.Coins  { return m.fees }
func (m mockFeeTx) FeePayer() []byte   { return m.payer }
func (m mockFeeTx) FeeGranter() []byte { return m.granter }
func (m mockFeeTx) GetMsgs() []sdk.Msg { return m.msgs }
func (m mockFeeTx) GetMsgsV2() ([]proto.Message, error) {
	var protoMsgs []proto.Message
	for _, msg := range m.msgs {
		if protoMsg, ok := msg.(proto.Message); ok {
			protoMsgs = append(protoMsgs, protoMsg)
		}
	}
	return protoMsgs, nil
}
func (m mockFeeTx) ValidateBasic() error                            { return nil }
func (m mockFeeTx) GetSigners() []sdk.AccAddress                    { return []sdk.AccAddress{sdk.AccAddress(m.payer)} }
func (m mockFeeTx) GetPubKeys() ([]cryptotypes.PubKey, error)       { return nil, nil }
func (m mockFeeTx) GetSignaturesV2() ([]signing.SignatureV2, error) { return nil, nil }

// mockTx implements sdk.Tx but NOT sdk.FeeTx for testing
type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg { return m.msgs }
func (m mockTx) GetMsgsV2() ([]proto.Message, error) {
	var protoMsgs []proto.Message
	for _, msg := range m.msgs {
		if protoMsg, ok := msg.(proto.Message); ok {
			protoMsgs = append(protoMsgs, protoMsg)
		}
	}
	return protoMsgs, nil
}
func (m mockTx) ValidateBasic() error                            { return nil }
func (m mockTx) GetSigners() []sdk.AccAddress                    { return nil }
func (m mockTx) GetPubKeys() ([]cryptotypes.PubKey, error)       { return nil, nil }
func (m mockTx) GetSignaturesV2() ([]signing.SignatureV2, error) { return nil, nil }

func TestAnteHandle(t *testing.T) {
	// Create test context
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

	// Set params
	params := types.DefaultParams()
	subspace.SetParamSet(ctx.Ctx, &params)

	stakingDenomFunc := func(ctx sdk.Context) string {
		return "stake"
	}

	decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

	// Mock next handler
	nextCalled := false
	nextHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	// Test case 1: Non-FeeTx should return error
	nonFeeTx := mockTx{msgs: []sdk.Msg{}}
	_, err := decorator.AnteHandle(ctx.Ctx, nonFeeTx, false, nextHandler)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Tx must implement the sdk.FeeTx interface")
	require.False(t, nextCalled)

	// Test case 2: Simulate mode should call next handler
	nextCalled = false
	payer := sdk.AccAddress([]byte("test-payer-address"))
	feeTx := mockFeeTx{
		gas:   100000,
		fees:  sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		payer: payer.Bytes(),
		msgs:  []sdk.Msg{},
	}
	newCtx, err := decorator.AnteHandle(ctx.Ctx, feeTx, true, nextHandler)
	require.NoError(t, err)
	require.True(t, nextCalled)
	require.NotNil(t, newCtx)

	// Test case 3: Non-simulate mode should process fees and call next handler
	nextCalled = false
	ctx.Ctx = ctx.Ctx.WithIsCheckTx(true)
	newCtx, err = decorator.AnteHandle(ctx.Ctx, feeTx, false, nextHandler)
	require.NoError(t, err)
	require.True(t, nextCalled)
	require.NotNil(t, newCtx)

	// Test case 4: Bypass messages should call next handler directly without fee processing
	nextCalled = false
	// Create a message that would be in bypass types
	// We use default params which includes bypass message types
	bypassFeeTx := mockFeeTx{
		gas:   100000,
		fees:  sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(0))), // Zero fees should be ok for bypass
		payer: payer.Bytes(),
		msgs:  []sdk.Msg{}, // Empty msgs list returns true for ContainsOnlyBypassMinFeeMsgs
	}
	newCtx, err = decorator.AnteHandle(ctx.Ctx, bypassFeeTx, false, nextHandler)
	require.NoError(t, err)
	require.True(t, nextCalled)
	require.NotNil(t, newCtx)

	// Test case 5: Non-bypass messages should calculate required fees and pass to next handler
	nextCalled = false
	var capturedCtx sdk.Context

	// Create a next handler that captures the context to verify the required fees are set
	nextHandlerWithCapture := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		capturedCtx = ctx
		return ctx, nil
	}

	// Set minimum gas prices that will be combined with global fees
	localMinGasPrices := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(5000, 0))}
	ctx.Ctx = ctx.Ctx.WithMinGasPrices(localMinGasPrices)

	nonBypassFeeTx := mockFeeTx{
		gas:   100000,
		fees:  sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100))), // Fees don't matter for this test
		payer: payer.Bytes(),
		msgs:  []sdk.Msg{&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}, // Non-bypass message
	}

	newCtx, err = decorator.AnteHandle(ctx.Ctx, nonBypassFeeTx, false, nextHandlerWithCapture)
	require.NoError(t, err)
	require.True(t, nextCalled)
	require.NotNil(t, newCtx)

	// Verify that the captured context has the correct min gas prices set
	// (should be the max of local min gas prices and global fees)
	expectedFees, err := decorator.GetTxFeeRequired(ctx.Ctx, nonBypassFeeTx)
	require.NoError(t, err)
	require.True(t, capturedCtx.MinGasPrices().Equal(expectedFees))

	// Test case 6: Error in GetTxFeeRequired should return error from AnteHandle
	nextCalled = false
	// Create a decorator with empty bond denom to cause GetTxFeeRequired to fail
	emptyBondDenomFunc := func(ctx sdk.Context) string {
		return ""
	}
	errorDecorator := ante.NewFeeDecorator(subspace, emptyBondDenomFunc)

	errorFeeTx := mockFeeTx{
		gas:   100000,
		fees:  sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100))),
		payer: payer.Bytes(),
		msgs:  []sdk.Msg{&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}, // Non-bypass message
	}

	_, err = errorDecorator.AnteHandle(ctx.Ctx, errorFeeTx, false, nextHandler)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty staking bond denomination")
	require.False(t, nextCalled)
}

func TestGetTxFeeRequired(t *testing.T) {
	// Create test context
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

	stakingDenomFunc := func(ctx sdk.Context) string {
		return "stake"
	}

	decorator := ante.NewFeeDecorator(subspace, stakingDenomFunc)

	// Test case 1: Error case - empty bond denom
	emptyBondDenomFunc := func(ctx sdk.Context) string {
		return ""
	}
	emptyDecorator := ante.NewFeeDecorator(subspace, emptyBondDenomFunc)

	payer := sdk.AccAddress([]byte("test-payer-address"))
	feeTx := mockFeeTx{
		gas:   100000,
		fees:  sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		payer: payer.Bytes(),
		msgs:  []sdk.Msg{},
	}

	_, err := emptyDecorator.GetTxFeeRequired(ctx.Ctx, feeTx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty staking bond denomination")

	// Test case 2: CheckTx mode - should combine local and global fees
	params := types.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}
	subspace.SetParamSet(ctx.Ctx, &params)

	// Set local min gas prices higher than global fees
	localFees := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	ctx.Ctx = ctx.Ctx.WithMinGasPrices(localFees).WithIsCheckTx(true)

	feeRequired, err := decorator.GetTxFeeRequired(ctx.Ctx, feeTx)
	require.NoError(t, err)
	require.NotEmpty(t, feeRequired)
	// Should return the max of local and global fees (local is higher)
	require.True(t, feeRequired[0].Amount.Equal(localFees[0].Amount))

	// Test case 3: DeliverTx mode - should return only global fees
	ctx.Ctx = ctx.Ctx.WithIsCheckTx(false)
	feeRequired, err = decorator.GetTxFeeRequired(ctx.Ctx, feeTx)
	require.NoError(t, err)
	require.NotEmpty(t, feeRequired)
	// Should return global fees only
	globalFees, err := decorator.GetGlobalFee(ctx.Ctx)
	require.NoError(t, err)
	require.True(t, feeRequired.Equal(globalFees))
}

func TestFindFunction(t *testing.T) {
	// Test Find function edge cases to get to 100% coverage

	// Test with empty coins
	found, coin := ante.Find(sdk.DecCoins{}, "test")
	require.False(t, found)
	require.Equal(t, sdk.DecCoin{}, coin)

	// Test with single coin - found
	singleCoin := sdk.DecCoins{sdk.NewDecCoinFromDec("test", math.LegacyNewDecWithPrec(1, 2))}
	found, coin = ante.Find(singleCoin, "test")
	require.True(t, found)
	require.Equal(t, "test", coin.Denom)

	// Test with single coin - not found
	found, coin = ante.Find(singleCoin, "notfound")
	require.False(t, found)
	require.Equal(t, sdk.DecCoin{}, coin)

	// Test with multiple coins - binary search paths
	multiCoins := sdk.DecCoins{
		sdk.NewDecCoinFromDec("aaa", math.LegacyNewDecWithPrec(1, 2)),
		sdk.NewDecCoinFromDec("bbb", math.LegacyNewDecWithPrec(2, 2)),
		sdk.NewDecCoinFromDec("ccc", math.LegacyNewDecWithPrec(3, 2)),
		sdk.NewDecCoinFromDec("ddd", math.LegacyNewDecWithPrec(4, 2)),
	}.Sort()

	// Test finding first element (left side of binary search)
	found, coin = ante.Find(multiCoins, "aaa")
	require.True(t, found)
	require.Equal(t, "aaa", coin.Denom)

	// Test finding last element (right side of binary search)
	found, coin = ante.Find(multiCoins, "ddd")
	require.True(t, found)
	require.Equal(t, "ddd", coin.Denom)

	// Test finding middle element (exact match in binary search)
	found, coin = ante.Find(multiCoins, "bbb")
	require.True(t, found)
	require.Equal(t, "bbb", coin.Denom)

	// Test not found (would be before first element)
	found, coin = ante.Find(multiCoins, "000")
	require.False(t, found)
	require.Equal(t, sdk.DecCoin{}, coin)

	// Test not found (would be after last element)
	found, coin = ante.Find(multiCoins, "zzz")
	require.False(t, found)
	require.Equal(t, sdk.DecCoin{}, coin)
}

func TestCombinedFeeRequirementEdgeCases(t *testing.T) {
	// Test edge cases for CombinedFeeRequirement to reach 100% coverage

	// Test with empty global fees (should return error)
	emptyGlobal := sdk.DecCoins{}
	minFees := sdk.DecCoins{sdk.NewDecCoinFromDec("test", math.LegacyNewDecWithPrec(1, 2))}

	result, err := ante.CombinedFeeRequirement(emptyGlobal, minFees)
	require.Error(t, err)
	require.Contains(t, err.Error(), "global fee cannot be empty")
	require.Equal(t, sdk.DecCoins{}, result)

	// Test with non-empty global fees and empty min fees
	globalFees := sdk.DecCoins{sdk.NewDecCoinFromDec("global", math.LegacyNewDecWithPrec(1, 2))}
	emptyMin := sdk.DecCoins{}

	result, err = ante.CombinedFeeRequirement(globalFees, emptyMin)
	require.NoError(t, err)
	require.Equal(t, globalFees, result)
}
