package keeper

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/cometbft/cometbft/libs/log"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

type Keeper struct {
	cdc                codec.BinaryCodec
	storeKey           storetypes.StoreKey
	paramSpace         paramtypes.Subspace
	bankKeeper         types.BankKeeper
	accountKeeper      types.AccountKeeper
	ContractOpsKeeper  wasmtypes.ContractOpsKeeper
	ContractViewKeeper wasmtypes.ViewKeeper
	AAKeeper           types.AbstractAccountKeeper

	// the address capable of executing a MsgSetPlatformPercentage message.
	// Typically, this should be the x/gov module account
	authority string
}

func NewKeeper(cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	paramSpace paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	wasmOpsKeeper wasmtypes.ContractOpsKeeper,
	wasmViewKeeper wasmtypes.ViewKeeper,
	aaKeeper types.AbstractAccountKeeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey:           key,
		cdc:                cdc,
		paramSpace:         paramSpace,
		bankKeeper:         bankKeeper,
		accountKeeper:      accountKeeper,
		ContractOpsKeeper:  wasmOpsKeeper,
		ContractViewKeeper: wasmViewKeeper,
		AAKeeper:           aaKeeper,
		authority:          authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdktypes.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// Platform Percentage

func (k Keeper) GetPlatformPercentage(ctx sdktypes.Context) math.Int {
	bz := ctx.KVStore(k.storeKey).Get(types.PlatformPercentageKey)
	percentage := sdktypes.BigEndianToUint64(bz)
	return math.NewIntFromUint64(percentage)
}

func (k Keeper) OverwritePlatformPercentage(ctx sdktypes.Context, percentage uint32) {
	ctx.KVStore(k.storeKey).Set(types.PlatformPercentageKey, sdktypes.Uint64ToBigEndian(uint64(percentage)))
}

// Authority

// GetAuthority returns the x/xion module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
