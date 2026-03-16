#!/bin/sh
set -eux

APPD=./build/xiond
SOURCE="$PWD/.testnode"
FORK="$PWD/.testnode-fork"

# Find the latest checkpoint
LATEST_CP=$(ls -d "$SOURCE/data/checkpoints/block_"* 2>/dev/null | sort -t_ -k2 -n | tail -1)
if [ -z "$LATEST_CP" ]; then
  echo "No checkpoints found in $SOURCE/data/checkpoints/"
  exit 1
fi
echo "Forking from checkpoint: $LATEST_CP"

# Copy config, keys, genesis (everything except data/)
rm -rf "$FORK"
mkdir -p "$FORK"
cp -r "$SOURCE/config" "$FORK/config"
cp -r "$SOURCE/keyring-test" "$FORK/keyring-test" 2>/dev/null || true

# Set up data dir from checkpoint contents
mkdir -p "$FORK/data"

# application.db — the app state checkpoint (hardlinked SSTs)
cp -r "$LATEST_CP/application.db" "$FORK/data/application.db"

# state.db — minimal CometBFT state (only keys needed to bootstrap)
cp -r "$LATEST_CP/state.db" "$FORK/data/state.db"

# blockstore.db — minimal block store (only last block's data)
cp -r "$LATEST_CP/blockstore.db" "$FORK/data/blockstore.db"

# priv_validator_state.json — reset so CometBFT can start signing
cp "$LATEST_CP/priv_validator_state.json" "$FORK/data/priv_validator_state.json"
sed -i 's/^timeout_propose = .*/timeout_propose = "3s"/' "$FORK/config/config.toml"
sed -i 's/^timeout_commit = .*/timeout_commit = "3s"/' "$FORK/config/config.toml"

# wasm cache — wasmvm needs the compiled contract artifacts
cp -r "$SOURCE/wasm" "$FORK/wasm" 2>/dev/null || true

# CometBFT will recreate evidence.db, tx_index.db, cs.wal fresh
$APPD start --home "$FORK"
