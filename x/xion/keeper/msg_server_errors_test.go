package keeper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

var (
	msgErrFrom = sdk.AccAddress("from_address_12345678")
	msgErrTo   = sdk.AccAddress("to_address_123456789")
)

func sendMsgFixture() *types.MsgSend {
	return &types.MsgSend{
		FromAddress: msgErrFrom.String(),
		ToAddress:   msgErrTo.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
	}
}

func multiSendMsgFixture(toAddr string) *types.MsgMultiSend {
	return &types.MsgMultiSend{
		Inputs: []banktypes.Input{{
			Address: msgErrFrom.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		}},
		Outputs: []banktypes.Output{{
			Address: toAddr,
			Coins:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		}},
	}
}

func setUxionMinimum(t *testing.T, ctx sdk.Context, keeper *Keeper) {
	t.Helper()
	require.NoError(t, keeper.OverwritePlatformMinimum(ctx, sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100)))))
}

func TestMsgServer_Send_CorruptMinimums(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	ctx.KVStore(keeper.storeKey).Set(types.PlatformMinimumKey, []byte("not-json"))
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	bank.On("BlockedAddr", msgErrTo).Return(false)

	_, err := server.Send(goCtx, sendMsgFixture())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid character")
	bank.AssertNotCalled(t, "SendCoins", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMsgServer_Send_UnconfiguredDenomRejected(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	setUxionMinimum(t, ctx, keeper)
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	bank.On("BlockedAddr", msgErrTo).Return(false)

	msg := sendMsgFixture()
	msg.Amount = sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(1000)))

	_, err := server.Send(goCtx, msg)
	require.ErrorIs(t, err, types.ErrMinimumNotMet)
}

func TestMsgServer_Send_PlatformFeeTransferFails(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	keeper.OverwritePlatformPercentage(ctx, 250)
	setUxionMinimum(t, ctx, keeper)
	boom := errors.New("fee collector unavailable")
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	bank.On("BlockedAddr", msgErrTo).Return(false)
	bank.On("SendCoinsFromAccountToModule", mock.Anything, msgErrFrom, mock.Anything, mock.Anything).Return(boom)

	_, err := server.Send(goCtx, sendMsgFixture())
	require.ErrorIs(t, err, boom)
	bank.AssertNotCalled(t, "SendCoins", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMsgServer_Send_TransferFails(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	setUxionMinimum(t, ctx, keeper)
	boom := errors.New("insufficient funds")
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	bank.On("BlockedAddr", msgErrTo).Return(false)
	bank.On("SendCoins", mock.Anything, msgErrFrom, msgErrTo, mock.Anything).Return(boom)

	_, err := server.Send(goCtx, sendMsgFixture())
	require.ErrorIs(t, err, boom)
}

func TestMsgServer_MultiSend_SendDisabled(t *testing.T) {
	goCtx, server, _, bank := setupMsgServerTest(t)

	disabled := errors.New("send disabled")
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(disabled)

	_, err := server.MultiSend(goCtx, multiSendMsgFixture(msgErrTo.String()))
	require.ErrorIs(t, err, disabled)
}

func TestMsgServer_MultiSend_CorruptMinimums(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	ctx.KVStore(keeper.storeKey).Set(types.PlatformMinimumKey, []byte("not-json"))
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)

	_, err := server.MultiSend(goCtx, multiSendMsgFixture(msgErrTo.String()))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid character")
}

func TestMsgServer_MultiSend_InvalidOutputAddress(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	setUxionMinimum(t, ctx, keeper)
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)

	_, err := server.MultiSend(goCtx, multiSendMsgFixture("not-a-bech32-address"))
	require.ErrorIs(t, err, sdkerrors.ErrInvalidAddress)
}

func TestMsgServer_MultiSend_FeeExceedsOutput(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	// The keeper store accepts any percentage; a value above 100% makes the
	// platform fee larger than the output it is taken from.
	keeper.OverwritePlatformPercentage(ctx, 20000)
	setUxionMinimum(t, ctx, keeper)
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	bank.On("BlockedAddr", msgErrTo).Return(false)

	_, err := server.MultiSend(goCtx, multiSendMsgFixture(msgErrTo.String()))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to subtract")
	bank.AssertNotCalled(t, "InputOutputCoins", mock.Anything, mock.Anything, mock.Anything)
}

func TestMsgServer_MultiSend_TransferFails(t *testing.T) {
	goCtx, server, keeper, bank := setupMsgServerTest(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	setUxionMinimum(t, ctx, keeper)
	boom := errors.New("insufficient funds")
	bank.On("IsSendEnabledCoins", mock.Anything, mock.Anything).Return(nil)
	bank.On("BlockedAddr", msgErrTo).Return(false)
	bank.On("InputOutputCoins", mock.Anything, mock.Anything, mock.Anything).Return(boom)

	_, err := server.MultiSend(goCtx, multiSendMsgFixture(msgErrTo.String()))
	require.ErrorIs(t, err, boom)
}

func TestMsgServer_SetPlatformPercentage_AboveMaximum(t *testing.T) {
	goCtx, server, keeper, _ := setupMsgServerTestWithAuthority(t, testAuthorityConst)
	ctx := sdk.UnwrapSDKContext(goCtx)

	_, err := server.SetPlatformPercentage(goCtx, &types.MsgSetPlatformPercentage{
		Authority:          testAuthorityConst,
		PlatformPercentage: 10001,
	})
	require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
	require.True(t, keeper.GetPlatformPercentage(ctx).IsZero())
}
