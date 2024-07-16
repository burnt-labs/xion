package types // noalias

import (
	"context"

	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"

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

type WasmKeeper interface {
	Migrate(ctx sdktypes.Context, contractAddress, caller sdktypes.AccAddress, newCodeID uint64, msg []byte) ([]byte, error)
	IterateContractsByCode(ctx sdktypes.Context, codeID uint64, cb func(address sdktypes.AccAddress) bool)
	PinCode(ctx sdktypes.Context, codeID uint64) error
	UnpinCode(ctx sdktypes.Context, codeID uint64) error
}

type AbstractAccountKeeper interface {
	GetParams(ctx sdktypes.Context) (*aatypes.Params, error)
	SetParams(ctx sdktypes.Context, params *aatypes.Params) error
}
