# Scripts

These scripts are copied from the [Cosmos-SDK](https://github.com/cosmos/cosmos-sdk/tree/v0.42.1/scripts) repository 
with minor modifications. All credits and big thanks go to the original authors.

Please note that a custom [fork](github.com/regen-network/protobuf) by the Regen network team is used.
See [`go.mod`](../go.mod) for version.

## Local v31 snapshot-conversion tooling

Build the previous-version binary from this branch, which supplies the
application-specific `newTestnetApp`/`InitForTestnet` callback:

```sh
bash scripts/build-in-place-testnet.sh
# Optional output path, configured in your environment:
bash scripts/build-in-place-testnet.sh "$PREV_BIN"
```

The helper targets SDK v0.53.8. It copies the SDK into a temporary directory
and applies `in-place-testnet-sdk.patch` there to construct complete replacement
commit signatures, including their block-ID flags. It does not edit the module
cache, `go.mod`, or `go.sum`. Use this helper to build the snapshot-conversion
binary until that SDK correction is included in the dependency itself.

Use `in-place-testnet` only on a disposable copy of node state, with fresh
validator and operator keys and a different chain ID:

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
