package types

import (
	"cosmossdk.io/collections"
)

// ParamsKey saves the current module params.
var (
	DkimPrefix = collections.NewPrefix(1)
)

const (
	ModuleName = "dkim"

	StoreKey = ModuleName

	QuerierRoute = ModuleName
)
