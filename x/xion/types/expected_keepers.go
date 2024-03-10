package types // noalias

import (
	"context"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// BankKeeper defines the contract needed to be fulfilled for banking and supply
// dependencies.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(
		ctx context.Context,
		senderModule string,
		recipientAddr sdktypes.AccAddress,
		amt sdktypes.Coins,
	) error

	SendCoinsFromModuleToModule(
		ctx context.Context,
		senderModule,
		recipientModule string,
		amt sdktypes.Coins,
	) error
	SendCoinsFromAccountToModule(
		ctx context.Context,
		senderAddr sdktypes.AccAddress,
		recipientModule string,
		amt sdktypes.Coins,
	) error
	IsSendEnabledCoins(ctx context.Context, coins ...sdktypes.Coin) error
	BlockedAddr(addr sdktypes.AccAddress) bool
	SendCoins(ctx context.Context, fromAddr sdktypes.AccAddress, toAddr sdktypes.AccAddress, amt sdktypes.Coins) error
	InputOutputCoins(ctx context.Context, input banktypes.Input, outputs []banktypes.Output) error
}

type AccountKeeper interface {
	GetModuleAccount(ctx context.Context, moduleName string) sdktypes.ModuleAccountI
}
