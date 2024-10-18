package types

var (
	PlatformPercentageKey = []byte{0x00}
	PlatformMinimumKey    = []byte{0x01}
)

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "xion"

	// StoreKey is the store key string for oracle
	StoreKey = ModuleName

	// RouterKey is the message route for oracle
	RouterKey = ModuleName

	// QuerierRoute is the querier route for oracle
	QuerierRoute = ModuleName
)
