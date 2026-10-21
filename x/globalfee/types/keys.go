package types

import "cosmossdk.io/collections"

const (
	// ModuleName is the name of the this module
	ModuleName = "globalfee"

	StoreKey = ModuleName

	QuerierRoute = ModuleName
)

var ParamsKey = collections.NewPrefix(0)
