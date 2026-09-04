package ibcwasm

import (
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers the 08-wasm client types so that records left in
// the IBC core store stay decodable. It registers nothing else: no Msg types, and
// no client route is added anywhere, so these clients can be read but never
// created or updated.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*exported.ClientState)(nil),
		&ClientState{},
	)
	registry.RegisterImplementations(
		(*exported.ConsensusState)(nil),
		&ConsensusState{},
	)
	registry.RegisterImplementations(
		(*exported.ClientMessage)(nil),
		&ClientMessage{},
	)
}
