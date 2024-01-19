package keeper

import (
	"github.com/burnt-labs/xion/x/jwk/types"
)

var _ types.QueryServer = Keeper{}
