package ibcwasm

import (
	"github.com/cosmos/ibc-go/v11/modules/core/exported"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the retired 08-wasm types so that records still
// on chain stay decodable: the client types left in the IBC core store, and the
// Msg types inside historical governance proposals. It registers implementations
// only. No Msg server and no client route is added anywhere, so these clients
// can be read but never created or updated, and these messages can be decoded
// but never executed.
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
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgStoreCode{},
		&MsgRemoveChecksum{},
		&MsgMigrateContract{},
	)
}
