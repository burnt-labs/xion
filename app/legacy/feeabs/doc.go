// Package feeabs keeps the fee abstraction types decodable after the module
// that used them was removed.
//
// Xion ran osmosis-labs/fee-abstraction as x/feeabs until v20 dropped its
// keeper, store and ante decorators. Governance proposals passed while it ran
// are still in state, and their messages are fee abstraction types
// (/xion.feeabs.v1beta1.MsgAddHostZone, MsgRemoveHostZone, MsgUpdateParams, and
// the /feeabstraction.feeabs.v1beta1.* URLs from before the proto package
// rename). The gov Proposals query decodes every proposal it pages through, so
// without these registrations one such proposal fails the whole listing.
//
// RegisterInterfaces and RegisterCodec register the types only. No keeper,
// store key, Msg server or gov route is wired, so the messages decode but can
// never execute. app/legacy_proposals_test.go checks that the type URLs found
// on xion-mainnet-1 still resolve.
//
// The protos remain in proto/xion/feeabs/v1beta1 with package
// xion.feeabs.v1beta1, so the type URLs are unchanged; only go_package points
// here.
package feeabs
