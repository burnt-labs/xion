// Package ibcwasm keeps the 08-wasm light-client types decodable after the
// module that defined them was removed.
//
// Xion dropped the optional IBC 08-wasm light client, along with its keeper,
// route, store key, and the burnt-labs/ibc-go fork that existed only to build
// that module against wasmvm v3. Deleting the module's own store does not touch
// the client records core IBC keeps under clients/08-wasm-N/ in the ibc store,
// and dropping the module also dropped the only RegisterInterfaces call that
// taught the codec about /ibc.lightclients.wasm.v1.*.
//
// xion-testnet-2 carries four such clients (08-wasm-11, -13, -14, -15, each
// wrapping a Parlia light client) together with their consensus states. Without
// a registration for those type URLs, 02-client ExportGenesis panics in
// MustUnmarshalClientState and the paginated ClientStates gRPC query fails
// outright rather than skipping the rows it cannot decode. xion-mainnet-1 has no
// 08-wasm records, but one binary serves both chains.
//
// So the concrete types are retained and nothing else is. RegisterInterfaces
// registers only ClientState, ConsensusState, and ClientMessage. There is
// deliberately no keeper, no store key, no client route, and no registration of
// MsgStoreCode, MsgMigrateContract, or MsgRemoveChecksum: the existing records
// decode, export, and validate, while no Wasm client can be created or updated.
//
// wasm.pb.go is vendored verbatim from ibc-go's
// modules/light-clients/08-wasm/v10@v10.5.0/types, with only the package clause
// changed. Its source, ibc/lightclients/wasm/v1/wasm.proto, is byte-identical at
// the ibc-go v10.7.0 tag this repository pins, and the messages are frozen wire
// formats. It is copied rather than imported because the 08-wasm types package
// pulls wasmvm in through wasm_vm.go, store.go, gas_register.go, and
// expected_interfaces.go — depending on it would drag the fork back in. It is not
// part of `make proto-gen` output; the proto lives in ibc-go.
package ibcwasm
