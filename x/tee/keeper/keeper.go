package keeper

import (
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/burnt-labs/xion/x/tee/types"
)

// Keeper defines the tee module keeper.
// This is a stateless module — no store, no authority, no params.
type Keeper struct {
	cdc    codec.BinaryCodec
	logger log.Logger
}

// NewKeeper creates a new tee module Keeper instance.
func NewKeeper(cdc codec.BinaryCodec, logger log.Logger) Keeper {
	return Keeper{
		cdc:    cdc,
		logger: logger.With(log.ModuleKey, "x/"+types.ModuleName),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger
}
