package types // noalias

import (
	"context"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// BankKeeper defines the contract needed to be fulfilled for banking and supply
// dependencies.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(
		ctx sdktypes.Context,
		senderModule string,
		recipientAddr sdktypes.AccAddress,
		amt sdktypes.Coins,
	) error
	SendCoinsFromModuleToModule(
		ctx sdktypes.Context,
		senderModule,
		recipientModule string,
		amt sdktypes.Coins,
	) error
	SendCoinsFromAccountToModule(
		ctx sdktypes.Context,
		senderAddr sdktypes.AccAddress,
		recipientModule string,
		amt sdktypes.Coins,
	) error
	Send(goCtx context.Context, msg *banktypes.MsgSend) (*banktypes.MsgSendResponse, error)
	MultiSend(goCtx context.Context, msg *banktypes.MsgMultiSend) (*banktypes.MsgMultiSendResponse, error)
}

type AccountKeeper interface {
	GetModuleAccount(ctx sdktypes.Context, moduleName string) authtypes.ModuleAccountI
}
