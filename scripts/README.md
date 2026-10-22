# Scripts

These scripts are copied from the [Cosmos-SDK](https://github.com/cosmos/cosmos-sdk/tree/v0.42.1/scripts) repository 
with minor modifications. All credits and big thanks go to the original authors.

Please note that a custom [fork](github.com/regen-network/protobuf) by the Regen network team is used.
See [`go.mod`](../go.mod) for version.

## Local v31 snapshot-conversion tooling

Build normally from this branch:

```sh
make build
```

Xion implements both application and consensus conversion in Go. The native
command constructs complete replacement commit signatures, updates validator
history and the cached genesis chain ID, and writes canonical CometBFT genesis
JSON before normal SDK startup. No SDK patch, custom build helper, dependency
fork, or special build flags are needed.

Use `in-place-testnet` only on a disposable copy of node state, with fresh
validator and operator keys and a different chain ID. The command refuses the
source chain ID, another revision of it, and any `xion-mainnet-N` or
`xion-testnet-N` chain ID: signatures are bound to the chain ID, so reusing a
public network's ID would make the copy's transactions and votes valid there.

```sh
"$PREV_BIN" in-place-testnet "$TEST_CHAIN_ID" "$OPERATOR_ADDRESS" \
  --home "$SCRATCH_HOME" --skip-confirmation
```

The application callback replaces active validator infrastructure, mints
synthetic validator stake and local operator funds, and shortens governance
voting periods to 20 seconds (10 seconds expedited). Existing validator records
and delegations are retained, but the replacement validator has synthetic shares
without a self-delegation record. Supply and staking invariants therefore do
not represent the source chain: use this clone for upgrade behavior, not supply
or staking-accounting comparisons.

Normal `start` still uses `newApp`. To resume a converted chain, use `start`
on the same home rather than converting it again. The optional
`--trigger-testnet-upgrade` schedules the named upgrade ten blocks after the
loaded height so conversion writes can commit first.

For a proposal-based rehearsal, omit the trigger flag: commit several v31
blocks, submit/deposit/vote for a software upgrade proposal, then hand the same
home to the candidate binary at the scheduled halt. The target-version branch
owns the Bash rehearsal runner; this branch only supplies the previous binary.

The binary must accept the snapshot's stored upgrade history. No upgrade
compatibility check is bypassed and no module version is rewritten.

Verified with an ordinary `make build`: persisted v31 state converted and
committed its first replacement-chain block at height 5, executed a signed bank
transfer, and resumed normal `start` through height 12. Module versions and
non-default globalfee/JWK parameters survived, with no manual signer or genesis
JSON rewrite. The same binary also completed the proposal-based v32 handoff:
proposal 1 passed, v32 applied at height 67, and blocks resumed through height 70.
This used disposable synthetic state, not a successful mainnet-archive migration.
