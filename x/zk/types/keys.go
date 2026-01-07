package types

import (
	"cosmossdk.io/collections"
)

// ParamsKey saves the current module params.
var (
	ParamsKey                = collections.NewPrefix(0)
	VKeyPrefix               = collections.NewPrefix(1)
	VkeySeqPrefix            = collections.NewPrefix(2)
	VkeyNameIndexPrefix      = collections.NewPrefix(3)
	VkeyAuthorityIndexPrefix = collections.NewPrefix(4)
)

const (
	ModuleName = "zk"

	StoreKey = ModuleName

	QuerierRoute = ModuleName
)
