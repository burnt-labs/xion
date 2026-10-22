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
validator and operator keys and a different chain ID:

```sh
"$PREV_BIN" in-place-testnet "$TEST_CHAIN_ID" "$OPERATOR_ADDRESS" \
  --home "$SCRATCH_HOME" --skip-confirmation
```

`--home` is required on the command line: the command does not fall back to
the default node home or to `XIOND_HOME`. It also refuses a home whose `priv_validator_laddr` is set. A
remote-signer validator (tmkms, horcrux) or a sentry keeps an unused stub key
in `config/`, which the key checks cannot tell apart from a fresh key, so copy
the data, clear `priv_validator_laddr` and generate a fresh local key first.
Before converting, the command prints the home and the chain ID in its genesis
file.

The application callback replaces active validator infrastructure, mints
synthetic validator stake and local operator funds, and shortens governance
voting periods to 20 seconds (10 seconds expedited). Existing validator records
and delegations are retained, but the replacement validator has synthetic shares
without a self-delegation record. Supply and staking invariants therefore do
not represent the source chain: use this clone for upgrade behavior, not supply
or staking-accounting comparisons. The source validators also keep status
`Bonded`; they only leave the power and last-validator indices. A delegation to
or undelegation from one of them puts it back in the power index, and the next
block adds it to the CometBFT validator set. Its power is negligible next to the
local validator, so the chain keeps running, but it misses every block and is
jailed and slashed for downtime: expect those jail and slash events in the
clone.

Normal `start` still uses `newApp`. To resume a converted chain, use `start`
on the same home rather than converting it again. The application half of the
conversion only commits with the first new block, while the CometBFT half is
written to disk before the node starts. If the command exits before that first
block commits (a busy port, a signal, a crash), plain `start` on that home halts
with `validator does not exist`: rerun the same `in-place-testnet` command, with
the same chain ID and keys, on the same home to finish the conversion. If
consensus state was only partly written, the command says the conversion was
interrupted and cannot be resumed; start again from a fresh copy. A rerun also
finishes the WAL and address-book reset if the earlier run stopped before it
completed. The optional
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
