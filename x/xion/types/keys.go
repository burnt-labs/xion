package types

import "cosmossdk.io/collections"

var (
	// ParamsKey saves the current module params.
	ParamsKey = collections.NewPrefix(0)
)

const (
	ModuleName = "xion"

	StoreKey = ModuleName

	QuerierRoute = ModuleName

	// RouterKey is the message route for oracle
	RouterKey = ModuleName
)
