# x/abstractaccount

Module that implements the `AbstractAccount` type and ante/post handler logic.

## Stable registration addresses

The module owns the stable account-address namespace while preserving
caller-selected, allowlisted account implementations:

- `address_derivation_hash` is the immutable 32-byte namespace used for every
  abstract-account address. It is address input, not executable code.
- `registration_enabled` lets governance pause new registrations without
  changing the namespace or blocking address queries.
- `MsgRegisterAccount.code_id` selects the contract code instantiated at the
  derived address. The existing allowlist still controls that selection.
- Registration calls the Wasm keeper's abstract-account-specific extension to
  instantiate the selected code directly at the address derived from
  `(address_derivation_hash, sender, salt)`.
- `AccountAddress(sender, salt)` returns the registered address or predicts that
  same stable address without depending on a code ID or checksum lookup.

There is no bootstrap contract or registration-time migration. Each selected
implementation only needs to accept its own instantiate message. Later contract
migrations continue to use the normal Wasm migration path.

The `(sender, salt) -> address` registry makes duplicate prevention a chain
invariant. Indexers may still provide history and analytics, but are not part
of registration correctness.

The v3 module migration leaves `address_derivation_hash` empty and
`registration_enabled` false. A chain upgrade must configure the chain-specific
hash and explicitly enable registration.

Genesis and chain upgrade handlers call the keeper's trusted `SetParams` path,
which can initialize the hash but does not enforce runtime immutability. An
upgrade must preserve an already configured hash. Changing it creates a
different address namespace and is unsupported.

AA-API and other clients should query `AccountAddress(sender, salt)` before
constructing address-bound authenticator credentials. Clients must not derive
correctness from the requested code ID or checksum.

Existing registration clients retain the same `MsgRegisterAccount` field
layout. The `code_id` continues to select the implementation, while the module
parameter independently controls the address.

## License

(c) larry0x, 2023 - [Apache 2.0](./LICENSE).
