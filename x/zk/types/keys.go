package types

import (
	"cosmossdk.io/collections"
)

// ParamsKey saves the current module params.
var (
	ParamsKey = collections.NewPrefix(0)
)

const (
	ModuleName = "zk"

	StoreKey = ModuleName

	QuerierRoute = ModuleName
)
