package keeper

import (
	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	paramSpace    paramtypes.Subspace
	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
}

func NewKeeper(cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	paramSpace paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		paramSpace:    paramSpace,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdktypes.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

////////////
// Params //
////////////

// GetParamSet returns the vote period from the parameters
func (k Keeper) GetParamSet(ctx sdktypes.Context) types.Params {
	var p types.Params
	k.paramSpace.GetParamSet(ctx, &p)
	return p
}

// setParams sets the parameters in the store
func (k Keeper) setParams(ctx sdktypes.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
