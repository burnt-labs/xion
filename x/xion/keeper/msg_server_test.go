package keeper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

const (
	testAuthorityConst = "test_authority"
)

// Mock bank keeper for testing
type MockBankKeeper struct {
	mock.Mock
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	args := m.Called(ctx, senderModule, recipientAddr, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	args := m.Called(ctx, senderModule, recipientModule, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	args := m.Called(ctx, senderAddr, recipientModule, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	args := m.Called(ctx, coins)
	return args.Error(0)
}

func (m *MockBankKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	args := m.Called(addr)
	return args.Bool(0)
}

func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	args := m.Called(ctx, fromAddr, toAddr, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) InputOutputCoins(ctx context.Context, input banktypes.Input, outputs []banktypes.Output) error {
	args := m.Called(ctx, input, outputs)
	return args.Error(0)
}

// Mock account keeper for testing
type MockAccountKeeper struct {
	mock.Mock
}

func (m *MockAccountKeeper) GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI {
	// Return a simple mock that implements the basic interface
	return authtypes.NewEmptyModuleAccount(moduleName, authtypes.Minter, authtypes.Burner)
}

func (m *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	args := m.Called(ctx, addr, denom)
	return args.Get(0).(sdk.Coin)
}

func (m *MockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	args := m.Called(ctx, moduleName, amt)
	return args.Error(0)
}

func setupMsgServerTestWithAuthority(t *testing.T, authority string) (context.Context, types.MsgServer, *Keeper, *MockBankKeeper) { // nolint:unparam (authority variations tested)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	mockBankKeeper := &MockBankKeeper{}

	keeper := Keeper{
		storeKey:   key,
		bankKeeper: mockBankKeeper,
		authority:  authority,
	}

	server := NewMsgServerImpl(keeper)

	return ctx, server, &keeper, mockBankKeeper
}

func setupMsgServerTest(t *testing.T) (context.Context, types.MsgServer, *Keeper, *MockBankKeeper) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	mockBankKeeper := &MockBankKeeper{}
	mockAccountKeeper := &MockAccountKeeper{}

	keeper := Keeper{
		storeKey:      key,
		bankKeeper:    mockBankKeeper,
		accountKeeper: mockAccountKeeper,
		authority:     "", // Initialize with empty authority, tests can set it explicitly
	}

	server := NewMsgServerImpl(keeper)

	return ctx, server, &keeper, mockBankKeeper
}

func TestNewMsgServerImpl(t *testing.T) {
	keeper := Keeper{}
	server := NewMsgServerImpl(keeper)
	require.NotNil(t, server)

	// Should return msgServer implementation
	_, ok := server.(*msgServer)
	require.True(t, ok)
}

func TestMsgServer_Send_Success(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Set up test data
	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")
	amount := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000)))

	msg := &types.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      amount,
	}

	// Set platform percentage to 2.5% (250 basis points)
	keeper.OverwritePlatformPercentage(ctx, 250)

	// Set platform minimums
	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	// Set up mock expectations
	mockBankKeeper.On("IsSendEnabledCoins", ctx, mock.AnythingOfType("[]types.Coin")).Return(nil)
	mockBankKeeper.On("BlockedAddr", toAddr).Return(false)

	// Calculate expected platform fee: 1000 * 250 / 10000 = 25
	platformFee := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(25)))
	throughAmount := amount.Sub(platformFee...)

	mockBankKeeper.On("SendCoinsFromAccountToModule", ctx, fromAddr, authtypes.FeeCollectorName, platformFee).Return(nil)
	mockBankKeeper.On("SendCoins", ctx, fromAddr, toAddr, throughAmount).Return(nil)

	// Execute
	response, err := server.Send(goCtx, msg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response)
	mockBankKeeper.AssertExpectations(t)
}

func TestMsgServer_Send_SendDisabled(t *testing.T) {
	goCtx, server, _, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")
	amount := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000)))

	msg := &types.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      amount,
	}

	// Mock send disabled error
	mockBankKeeper.On("IsSendEnabledCoins", ctx, mock.AnythingOfType("[]types.Coin")).Return(banktypes.ErrSendDisabled)

	// Execute
	_, err := server.Send(goCtx, msg)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, banktypes.ErrSendDisabled)
}

func TestMsgServer_Send_InvalidFromAddress(t *testing.T) {
	goCtx, server, _, mockBankKeeper := setupMsgServerTest(t)

	msg := &types.MsgSend{
		FromAddress: "invalid_address",
		ToAddress:   "to_address",
		Amount:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
	}

	// Set up mock - the Send method will call IsSendEnabledCoins first
	mockBankKeeper.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)

	// Execute
	_, err := server.Send(goCtx, msg)

	// Assert - should fail due to invalid address parsing
	require.Error(t, err)
	require.Contains(t, err.Error(), "decoding bech32 failed")
}

func TestMsgServer_Send_InvalidToAddress(t *testing.T) {
	goCtx, server, _, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	amount := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000)))

	msg := &types.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   "invalid_address",
		Amount:      amount,
	}

	mockBankKeeper.On("IsSendEnabledCoins", ctx, mock.AnythingOfType("[]types.Coin")).Return(nil)

	// Execute
	_, err := server.Send(goCtx, msg)

	// Assert
	require.Error(t, err)
}

func TestMsgServer_Send_BlockedAddress(t *testing.T) {
	goCtx, server, _, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")
	amount := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000)))

	msg := &types.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      amount,
	}

	mockBankKeeper.On("IsSendEnabledCoins", ctx, mock.AnythingOfType("[]types.Coin")).Return(nil)
	mockBankKeeper.On("BlockedAddr", toAddr).Return(true)

	// Execute
	_, err := server.Send(goCtx, msg)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not allowed to receive funds")
}

func TestMsgServer_Send_MinimumNotMet(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")
	amount := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(50))) // Less than minimum

	msg := &types.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      amount,
	}

	// Set platform minimums higher than the amount
	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	mockBankKeeper.On("IsSendEnabledCoins", ctx, mock.AnythingOfType("[]types.Coin")).Return(nil)
	mockBankKeeper.On("BlockedAddr", toAddr).Return(false)

	// Execute
	_, err = server.Send(goCtx, msg)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "minimum send amount not met")
}

func TestMsgServer_Send_EmptyMinimumsNotMet(t *testing.T) {
	goCtx, server, _, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")
	amount := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))

	msg := &types.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      amount,
	}

	// No platform minimums set (empty by default)
	// This should fail because platform minimums must be explicitly configured

	mockBankKeeper.On("IsSendEnabledCoins", ctx, mock.AnythingOfType("[]types.Coin")).Return(nil)
	mockBankKeeper.On("BlockedAddr", toAddr).Return(false)

	// Execute
	_, err := server.Send(goCtx, msg)

	// Assert - should fail when no minimums are configured
	require.Error(t, err)
	require.Contains(t, err.Error(), "minimum send amount not met")
}

func TestMsgServer_Send_ZeroPercentage(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")
	amount := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000)))

	msg := &types.MsgSend{
		FromAddress: fromAddr.String(),
		ToAddress:   toAddr.String(),
		Amount:      amount,
	}

	// Set platform percentage to 0
	keeper.OverwritePlatformPercentage(ctx, 0)

	// Set platform minimums
	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	mockBankKeeper.On("IsSendEnabledCoins", ctx, mock.AnythingOfType("[]types.Coin")).Return(nil)
	mockBankKeeper.On("BlockedAddr", toAddr).Return(false)
	// No platform fee should be deducted when percentage is 0
	mockBankKeeper.On("SendCoins", ctx, fromAddr, toAddr, amount).Return(nil)

	// Execute
	response, err := server.Send(goCtx, msg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response)
	mockBankKeeper.AssertExpectations(t)
}

func TestMsgServer_MultiSend_Success(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr1 := sdk.AccAddress("to_address_1234567890")
	toAddr2 := sdk.AccAddress("to_address_2345678901")

	inputs := []banktypes.Input{{
		Address: fromAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(2000))),
	}}

	outputs := []banktypes.Output{
		{
			Address: toAddr1.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		},
		{
			Address: toAddr2.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		},
	}

	msg := &types.MsgMultiSend{
		Inputs:  inputs,
		Outputs: outputs,
	}

	// Setup platform fees - set percentage to 0 for simpler test
	keeper.OverwritePlatformPercentage(ctx, 0)

	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	// Set up mocks
	mockBankKeeper.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	mockBankKeeper.On("BlockedAddr", mock.Anything).Return(false)
	mockBankKeeper.On("InputOutputCoins", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Execute
	response, err := server.MultiSend(goCtx, msg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestMsgServer_MultiSend_MinimumNotMet(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")

	inputs := []banktypes.Input{{
		Address: fromAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(50))), // Below minimum
	}}

	outputs := []banktypes.Output{{
		Address: toAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(50))),
	}}

	msg := &types.MsgMultiSend{
		Inputs:  inputs,
		Outputs: outputs,
	}

	// Set platform minimums higher than the amount
	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	// Set up mocks
	mockBankKeeper.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)

	// Execute
	response, err := server.MultiSend(goCtx, msg)

	// Assert
	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "minimum send amount not met")
}

func TestMsgServer_SetPlatformPercentage_Success(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTestWithAuthority(t, testAuthorityConst)
	ctx := sdk.UnwrapSDKContext(goCtx)
	require.NotNil(t, mockBankKeeper)

	msg := &types.MsgSetPlatformPercentage{
		Authority:          testAuthorityConst,
		PlatformPercentage: 500, // 5%
	}

	// Execute
	response, err := server.SetPlatformPercentage(goCtx, msg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify the percentage was set
	storedPercentage := keeper.GetPlatformPercentage(ctx).Uint64()
	require.Equal(t, uint64(500), storedPercentage)
}

func TestMsgServer_SetPlatformPercentage_InvalidAuthority(t *testing.T) {
	goCtx, server, keeper, _ := setupMsgServerTest(t)

	// Set up the authority in the keeper
	keeper.authority = "correct_authority"

	msg := &types.MsgSetPlatformPercentage{
		Authority:          "wrong_authority",
		PlatformPercentage: 500,
	}

	// Execute
	response, err := server.SetPlatformPercentage(goCtx, msg)

	// Assert
	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "invalid authority")
}

func TestMsgServer_SetPlatformMinimum_Success(t *testing.T) {
	goCtx, server, keeper, _ := setupMsgServerTestWithAuthority(t, testAuthorityConst)
	ctx := sdk.UnwrapSDKContext(goCtx)

	testMinimums := sdk.NewCoins(
		sdk.NewCoin("uxion", math.NewInt(200)),
		sdk.NewCoin("utest", math.NewInt(100)),
	)

	msg := &types.MsgSetPlatformMinimum{
		Authority: testAuthorityConst,
		Minimums:  testMinimums,
	}

	// Execute
	response, err := server.SetPlatformMinimum(goCtx, msg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify the minimums were set
	storedMinimums, err := keeper.GetPlatformMinimums(ctx)
	require.NoError(t, err)
	require.True(t, testMinimums.Equal(storedMinimums))
}

func TestMsgServer_SetPlatformMinimum_InvalidAuthority(t *testing.T) {
	goCtx, server, keeper, _ := setupMsgServerTest(t)

	// Set up the authority in the keeper
	keeper.authority = "correct_authority"

	testMinimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))

	msg := &types.MsgSetPlatformMinimum{
		Authority: "wrong_authority",
		Minimums:  testMinimums,
	}

	// Execute
	response, err := server.SetPlatformMinimum(goCtx, msg)

	// Assert
	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "invalid authority")
}

func TestMsgServer_MultiSend_BlockedAddress(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	blockedAddr := sdk.AccAddress("blocked_address_123456")

	inputs := []banktypes.Input{{
		Address: fromAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
	}}

	outputs := []banktypes.Output{{
		Address: blockedAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
	}}

	msg := &types.MsgMultiSend{
		Inputs:  inputs,
		Outputs: outputs,
	}

	// Set platform minimums to pass minimum check
	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	// Set up mocks - blocked address returns true
	mockBankKeeper.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	mockBankKeeper.On("BlockedAddr", blockedAddr).Return(true)

	// Execute
	response, err := server.MultiSend(goCtx, msg)

	// Assert
	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "is not allowed to receive funds")
}

func TestMsgServer_MultiSend_WithPlatformFee(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")

	inputs := []banktypes.Input{{
		Address: fromAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
	}}

	outputs := []banktypes.Output{{
		Address: toAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
	}}

	msg := &types.MsgMultiSend{
		Inputs:  inputs,
		Outputs: outputs,
	}

	// Set platform percentage to 5%
	keeper.OverwritePlatformPercentage(ctx, 500) // 5% = 500 basis points

	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(10)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	// Set up mocks
	mockBankKeeper.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	mockBankKeeper.On("BlockedAddr", mock.Anything).Return(false)
	mockBankKeeper.On("InputOutputCoins", mock.Anything, mock.Anything, mock.MatchedBy(func(outputs []banktypes.Output) bool {
		// Should have 2 outputs: recipient + fee collector
		return len(outputs) == 2
	})).Return(nil)

	// Execute
	response, err := server.MultiSend(goCtx, msg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response)
	mockBankKeeper.AssertExpectations(t)
}

func TestMsgServer_MultiSend_HighPlatformFee(t *testing.T) {
	goCtx, server, keeper, mockBankKeeper := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_123456789")

	inputs := []banktypes.Input{{
		Address: fromAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100))),
	}}

	outputs := []banktypes.Output{{
		Address: toAddr.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100))),
	}}

	msg := &types.MsgMultiSend{
		Inputs:  inputs,
		Outputs: outputs,
	}

	// Set high platform percentage
	keeper.OverwritePlatformPercentage(ctx, 9000) // 90%

	minimums := sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1)))
	err := keeper.OverwritePlatformMinimum(ctx, minimums)
	require.NoError(t, err)

	// Set up mocks
	mockBankKeeper.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	mockBankKeeper.On("BlockedAddr", mock.Anything).Return(false)
	mockBankKeeper.On("InputOutputCoins", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Execute
	response, err := server.MultiSend(goCtx, msg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestGetPlatformCoins(t *testing.T) {
	tests := []struct {
		name       string
		coins      sdk.Coins
		percentage math.Int
		expected   sdk.Coins
	}{
		{
			name:       "zero percentage",
			coins:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
			percentage: math.NewInt(0),
			expected:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(0))),
		},
		{
			name:       "10% of 1000",
			coins:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
			percentage: math.NewInt(1000), // 10% in basis points
			expected:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100))),
		},
		{
			name:       "50% of 2000",
			coins:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(2000))),
			percentage: math.NewInt(5000), // 50% in basis points
			expected:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		},
		{
			name:       "multiple coins",
			coins:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000)), sdk.NewCoin("uatom", math.NewInt(500))),
			percentage: math.NewInt(1000), // 10%
			expected:   sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(50)), sdk.NewCoin("uxion", math.NewInt(100))),
		},
		{
			name:       "100% fee",
			coins:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
			percentage: math.NewInt(10000), // 100%
			expected:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		},
		{
			name:       "very small amount",
			coins:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1))),
			percentage: math.NewInt(1000), // 10%
			expected:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(0))), // Rounds down
		},
		{
			name:       "empty coins",
			coins:      sdk.NewCoins(),
			percentage: math.NewInt(1000),
			expected:   sdk.NewCoins(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPlatformCoins(tt.coins, tt.percentage)
			require.True(t, tt.expected.Equal(result),
				"expected %s, got %s", tt.expected, result)
		})
	}
}

func TestGetPlatformCoins_LargeAmounts(t *testing.T) {
	// Test the big integer arithmetic path for very large amounts
	tests := []struct {
		name       string
		amount     string
		percentage math.Int
		expectedFn func(amount math.Int) math.Int
	}{
		{
			name:       "very large amount - normal path",
			amount:     "1000000000000000000", // 1e18
			percentage: math.NewInt(1000),     // 10%
			expectedFn: func(amount math.Int) math.Int {
				return amount.Mul(math.NewInt(1000)).Quo(math.NewInt(10000))
			},
		},
		{
			name:       "extremely large amount - big int path",
			amount:     "999999999999999999999999999999999999999999", // Very large number
			percentage: math.NewInt(1000), // 10%
			expectedFn: func(amount math.Int) math.Int {
				// Should use big integer arithmetic
				return amount.Mul(math.NewInt(1000)).Quo(math.NewInt(10000))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, ok := math.NewIntFromString(tt.amount)
			require.True(t, ok, "failed to parse amount")

			coins := sdk.NewCoins(sdk.NewCoin("uxion", amount))
			result := getPlatformCoins(coins, tt.percentage)

			expected := tt.expectedFn(amount)
			expectedCoins := sdk.NewCoins(sdk.NewCoin("uxion", expected))

			require.True(t, expectedCoins.Equal(result),
				"expected %s, got %s", expectedCoins, result)
		})
	}
}
