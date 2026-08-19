# Abstract-account fixed address hash in v31

## Outcome

Integrate the keeper-only Wasmd fixed-hash instantiation extension and the
abstract-account module's direct-instantiation design into `release/v31`.

## Upgrade behavior

- Run module migrations before configuring the new parameter.
- Configure the mainnet namespace with the live code ID 5 checksum.
- Configure the testnet namespace with the live code ID 1 checksum.
- Enable registration only for the explicitly supported chain IDs.
- Leave unknown chains unconfigured and registration-disabled.
- Reject an already configured conflicting hash rather than changing the
  address namespace.

## Verification

- Confirm both constants against the live chain Wasm code queries.
- Test mainnet and testnet configuration, idempotence, unsupported-chain
  behavior, and conflicting namespace rejection.
- Run focused upgrade tests, full unit tests, lint, and build gates.
