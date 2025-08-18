package types

import (
	"context"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	IterateAccounts(ctx context.Context, process func(sdk.AccountI) (stop bool))
	SetModuleAccount(context.Context, sdk.ModuleAccountI)
	GetParams(ctx context.Context) authtypes.Params
	SetAccount(ctx context.Context, acc sdk.AccountI)
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
}

type FeegrantKeeper interface {
	UseGrantedFees(ctx context.Context, granter, grantee sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) error
}

// StakingKeeper define the expected interface to retrieve staking denom
type StakingKeeper interface {
	// BondDenom - Bondable coin denomination
	BondDenom(ctx context.Context) (res string, err error)
}

// ClientKeeper defines the expected IBC client keeper.
type ClientKeeper interface {
	GetClientConsensusState(ctx context.Context, clientID string) (connection ibcexported.ConsensusState, found bool)
}

// ConnectionKeeper defines the expected IBC connection keeper.
type ConnectionKeeper interface {
	GetConnection(ctx context.Context, connectionID string) (connection connectiontypes.ConnectionEnd, found bool)
}

// PortKeeper defines the expected IBC port keeper.
type PortKeeper interface {
	BindPort(ctx context.Context, portID string) *capabilitytypes.Capability
}

// ScopedKeeper defines the expected scoped keeper.
type ScopedKeeper interface {
	AuthenticateCapability(ctx context.Context, capability *capabilitytypes.Capability, name string) bool
	ClaimCapability(ctx context.Context, capability *capabilitytypes.Capability, name string) error
	GetCapability(ctx context.Context, name string) (*capabilitytypes.Capability, bool)
}

// ChannelKeeper defines the expected IBC channel keeper.
type ChannelKeeper interface {
	GetChannel(ctx context.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSend(ctx context.Context, portID, channelID string) (uint64, bool)
	ChanCloseInit(ctx context.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error
}
