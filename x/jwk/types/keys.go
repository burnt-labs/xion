package types

const (
	// ModuleName defines the module name
	ModuleName = "jwk"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

var ParamsKey = []byte("Params")

func KeyPrefix(p string) []byte {
	return []byte(p)
}
