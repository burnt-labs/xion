package types

import (
	"cosmossdk.io/collections"
)

// ParamsKey saves the current module params.
var (
	ParamsKey  = collections.NewPrefix(0)
	DkimPrefix = collections.NewPrefix(1)
)

const (
	ModuleName = "dkim"

	StoreKey = ModuleName

	QuerierRoute = ModuleName
)
