package types // noalias

import (
	"context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
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
	IsSendEnabledCoins(ctx sdktypes.Context, coins ...sdktypes.Coin) error
	BlockedAddr(addr sdktypes.AccAddress) bool
	SendCoins(ctx sdktypes.Context, fromAddr sdktypes.AccAddress, toAddr sdktypes.AccAddress, amt sdktypes.Coins) error
	InputOutputCoins(ctx sdktypes.Context, inputs []banktypes.Input, outputs []banktypes.Output) error
}

type AccountKeeper interface {
	GetModuleAccount(ctx sdktypes.Context, moduleName string) authtypes.ModuleAccountI
}

type WasmKeeper interface {
	GetCodeInfo(ctx sdktypes.Context, codeID uint64) *wasmtypes.CodeInfo
	GetContractInfo(ctx sdktypes.Context, contractAddress sdktypes.AccAddress) *wasmtypes.ContractInfo
}

type AuthzKeeper interface {
	Grants(c context.Context, req *authztypes.QueryGrantsRequest) (*authztypes.QueryGrantsResponse, error)
	DeleteGrant(ctx sdktypes.Context, grantee sdktypes.AccAddress, granter sdktypes.AccAddress, msgType string) error
	DispatchActions(ctx sdktypes.Context, grantee sdktypes.AccAddress, msgs []sdktypes.Msg) ([][]byte, error)
}
