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

Use only a disposable copy of persisted state, fresh validator/operator keys,
and a different chain ID. The initializer replaces active validator indices,
initializes distribution/slashing records, mints synthetic stake and local
operator funds, and shortens governance voting periods to 20 seconds
(10 seconds expedited). Existing validator/delegation records remain, while
the replacement validator has synthetic shares without a self-delegation.
Supply and staking invariants are not a replica of the source chain.

Resume a converted chain with normal `start` on the same home. Do not convert
the same home twice. The optional `--trigger-testnet-upgrade` schedules the named
upgrade ten blocks after the loaded height so conversion writes commit first.
Stored module versions and upgrade-compatibility checks are not bypassed.

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
