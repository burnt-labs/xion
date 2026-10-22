# Scripts

These scripts are copied from the [Cosmos-SDK](https://github.com/cosmos/cosmos-sdk/tree/v0.42.1/scripts) repository 
with minor modifications. All credits and big thanks go to the original authors.

Please note that a custom [fork](github.com/regen-network/protobuf) by the Regen network team is used.
See [`go.mod`](../go.mod) for version.

## In-place conversion of v32 state

The `in-place-testnet` command uses a dedicated application initializer.
Ordinary `start`, export, snapshots and indexer commands keep the normal
application constructor.

Build normally; no SDK patch, custom build helper, or dependency fork is needed:

```bash
make build
build/xiond in-place-testnet "$TEST_CHAIN_ID" "$OPERATOR_ADDRESS" \
  --home "$SCRATCH_HOME" --skip-confirmation
```

Xion owns consensus conversion in Go: it constructs complete replacement commit
signatures, updates validator history and the cached genesis chain ID, and saves
canonical CometBFT genesis JSON. The command starts through the normal SDK
server path rather than SDK `testnetify`. This builds a **v32** converter, not
the v31 binary needed for the v31-to-v32 rehearsal below.

Use only a disposable copy of persisted state and a fresh operator key.
Protections, in order:

1. **Fresh validator key per copy.** The command generates a new consensus key
   for every conversion; the copy's only validator signs with it. The home's
   existing `priv_validator_key.json` and `priv_validator_state.json` are moved
   to `*.pre-testnet` inside the home (the first backup is kept) and never used
   for signing, so a copy cannot double-sign with a live validator's key under
   any chain ID. The new consensus address is printed before conversion.
2. **Not the source chain.** The new chain ID must differ from the source chain
   ID and must not be another revision of it (IBC treats `foo-2` as `foo-1`
   upgraded).
3. **Not a public network name.** Any `xion-mainnet-N` or `xion-testnet-N`
   chain ID is refused as a precaution against transactions that could be
   replayed on those networks.

`--home` must be given on the command line: the command does not fall back to
the default node home or to `XIOND_HOME`. A home with `priv_validator_laddr`
set is refused: a remote-signer validator (tmkms, horcrux) or a sentry keeps an
unused stub key in `config/`, so copy the data and clear `priv_validator_laddr`
first. Nothing is written if a check fails or the prompt is declined. The
initializer replaces active validator indices, initializes distribution/slashing
records, mints synthetic stake and local operator funds, and shortens governance
voting periods to 20 seconds (10 seconds expedited). Existing
validator/delegation records remain, while the replacement validator has
synthetic shares without a self-delegation. Supply and staking invariants are
not a replica of the source chain. The source validators also keep status
`Bonded`; they only leave the power and last-validator indices. A delegation to
or undelegation from one of them puts it back in the power index, and the next
block adds it to the CometBFT validator set. Its power is negligible next to the
local validator, so the chain keeps running, but it misses every block and is
jailed and slashed for downtime: expect those events in the copy.

`--grpc-only` is refused: without consensus no block commits, and the conversion
only completes with the first new block.

Resume a converted chain with normal `start` on the same home. Do not convert
the same home twice. The application half of a conversion only commits with the
first new block, while the CometBFT half is written before the node starts. If
the command exits before that first block commits (a busy port, a signal, a
crash), plain `start` halts with `validator does not exist`: rerun the same
`in-place-testnet` command, with the same chain ID, on the same home. The rerun
keeps the validator key the interrupted conversion installed (it is used only
when the home holds the `*.pre-testnet` key backup, which is written empty
when the home had no key, and the converted consensus set
is exactly that key at the testnet power) and finishes the WAL and address-book
reset if it was cut short. If consensus state was only partly written, the
command says the conversion cannot be resumed; start again from a fresh copy.
The optional `--trigger-testnet-upgrade` schedules the named upgrade ten blocks
after the loaded height so conversion writes commit first. Stored module
versions and upgrade-compatibility checks are not bypassed.

Native conversion was exercised on persisted v32 state using the normal build:
the converted chain committed a signed bank transfer and resumed ordinary
`start` through height 79. Module versions and non-default globalfee/JWK
parameters survived. The restored source signing watermark was left intact
for the command to handle; no manual signer or genesis JSON rewrite was used.

## Rehearse v31-to-v32 from this checkout

Build the previous-version conversion binary from the v31 implementation in
[PR #667](https://github.com/burnt-labs/xion/pull/667), using normal `make build`.
The previous binary must accept the snapshot's stored upgrade history; CLI
registration alone is insufficient.

Configure paths privately in your shell or `.envrc`:

```bash
export PREV_BIN="/absolute/path/to/xiond-v31"
export SNAPSHOT="/absolute/path/to/snapshot.tar.lz4"
unset CUR_BIN

bash scripts/rehearse-in-place-testnet.sh
```

`PREV_BIN` and `SNAPSHOT` are required. By default, the Bash runner first invokes
the existing `make build` in **its own checkout** and uses the resulting
`build/xiond`. An existing binary does not skip the build. Set `CUR_BIN` or pass
`--cur-bin PATH` only to explicitly select a prebuilt candidate. No Makefile
changes or commit-named binaries are required.

Optional `GENESIS` supplies a source genesis file. Optional `WORK_DIR` selects a
new scratch directory; otherwise, a unique sibling of the archive is used.
CLI arguments override environment variables. The script does not source
`.envrc`; load it through your shell or direnv. `.envrc` and `.direnv/` are
ignored by Git.

The runner initializes fresh keys/config before installing the archive's
`data` and `wasm`. All CLI and daemon commands use the explicit scratch home,
not the default node home. Archived validator credentials/config are not
installed, the archive is read-only, listeners are loopback-only, peers are
disabled, and API/gRPC listeners are disabled.

It converts using v31, commits several blocks, submits/deposits/votes for an
upgrade proposal, requires that proposal to pass and the exact scheduled halt,
then starts the v32 candidate on the **same home**. No direct trigger flag,
bridge migration, module-version rewriting, or upgrade-guard bypass is used.

The script refuses to start if anything listens on the RPC port or the P2P
port below it, and refuses `--source-chain xion-testnet-2` for v32. The v32
upgrade registers the checksum-derived abstract accounts of the chain IDs in
`checksumAccountCodes` (`app/upgrades.go`) and looks them up by the chain ID
the upgrade runs under. A rehearsal runs under its own chain ID, so a testnet-2
snapshot would skip that backfill and still report success. Mainnet, the
default, has no such entries.

Success requires resumed blocks, the exact applied upgrade height, expected
v32 module versions, unmounted retired stores, and preserved globalfee/JWK
parameters. Historical version-map entries can remain after store removal.
Logs, JSON evidence and disposable keys/state are retained under the work
directory. `result.json` is written only after all checks pass; child daemons
are stopped on exit.

The fresh-build Bash path was exercised end-to-end on disposable v31 state:
proposal 1 passed, v32 applied at height 67, and blocks resumed through height 70.
The module-version, retired-store, and non-default parameter checks passed.
This is synthetic-fixture proof, not a successful mainnet-archive rehearsal;
an incompatible stored upgrade history remains a blocker.
