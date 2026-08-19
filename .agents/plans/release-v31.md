# Release v31

Integration state for the `v31` upgrade. Supersedes the individual plans in
this directory for anything they disagree on.

## Contents

1. **Barretenberg v5.** `burnt-labs/barretenberg-go` moves off the
   `v5.0.0-rc.1` candidate onto Aztec `v5.2.0`. The release changed the
   UltraHonk transcript, so the `x/zk` and e2e test vectors are regenerated.
2. **No forked IBC.** The optional `08-wasm` light client is removed, which
   drops the last `cosmos/ibc-go` fork from the module graph. The legacy store
   is deleted during the `v31` upgrade.
3. **Stable abstract account addresses.** The wasmd fork gains a keeper-only
   `Instantiate2WithAddressHash`, and the upgrade pins each supported chain's
   address namespace to the checksum already in use there.
4. **Current core stack.** Cosmos SDK, wasmd, CometBFT, IBC, and wasmvm move to
   the current patch releases of the lines already in use, and every fork pin
   becomes a released tag rather than a pseudo-version.

## Fork releases this depends on

| Module | Version | Contains |
|---|---|---|
| `burnt-labs/wasmd` | `v0.61.14-xion.1` | keeper-only fixed address hash, rebased on wasmd `v0.61.14` |
| `burnt-labs/abstract-account` | `v0.1.8` | chain-owned account addresses |
| `burnt-labs/barretenberg-go` | Aztec `v5.2.0` | UltraHonk verification against the stable release |

`abstract-account` skips `v0.1.6` and `v0.1.7`. The Go module proxy holds
immutable entries for both, cut from commits that no longer exist in the
repository, so neither version can ever resolve to current code.

## Address namespace

The upgrade configures `AddressDerivationHash` per chain and enables
registration only where it is configured:

| Chain | Code ID | Checksum |
|---|---|---|
| `xion-mainnet-1` | 5 | `FEFA4D0C…AAF17D1B` |
| `xion-testnet-2` | 1 | `FC06F022…24567DB8` |

Both were confirmed against the live chains' Wasm code queries. Each is the
checksum that already derived that chain's existing abstract account addresses,
so pinning it keeps every existing account at its current address while
decoupling future addresses from whatever code they instantiate. Unsupported
chains stay unconfigured and registration-disabled, and a chain already carrying
a different hash is rejected rather than silently renamespaced.

## Operational note: 08-wasm on testnet

Mainnet has no `08-wasm` light clients, so the store deletion is clean there.

`xion-testnet-2` has four: `08-wasm-11`, `08-wasm-13`, `08-wasm-14`, and
`08-wasm-15`. Only `08-wasm-15` backs anything live — a Parlia (BSC) client on
`connection-15` carrying transfer `channel-8` in `STATE_OPEN`, holding 2100
uxion of escrow and four in-flight packet commitments. The client is stale
(latest height 196). Removing the module strands that channel, its escrow, and
those packets permanently. Confirm the BSC integration is not coming back before
the testnet upgrade, and drain or close the channel first if it is.

## Verification

- Root and e2e modules build; `go vet ./...` clean in both.
- Full unit suite passes.
- Address derivation constants confirmed against live mainnet and testnet.
- Barretenberg vectors regenerated and verified on linux amd64, arm64, and
  arm64-musl.

Two unit tests fail only outside CI: `x/dkim/client/cli TestGetDkimDNSPublicKey`
needs live DNS, and the `client/cli` suites misbehave when a local `xiond` is
listening on 26657, because they assume queries cannot reach a node.
