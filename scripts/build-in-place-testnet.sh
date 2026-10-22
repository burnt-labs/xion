#!/usr/bin/env bash
# Local tooling build with the SDK commit-signature correction required by
# snapshots whose first validator was absent from the last commit.
set -Eeuo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd -- "$ROOT"
OUTPUT=${1:-"$ROOT/build/xiond"}
[[ $# -le 1 ]] || { echo "Usage: $0 [output-binary]" >&2; exit 2; }
for tool in go jq patch; do command -v "$tool" >/dev/null || { echo "Missing $tool" >&2; exit 1; }; done
SDK_INFO=$(GOWORK=off go mod download -json github.com/cosmos/cosmos-sdk)
[[ $(jq -r '.Version' <<<"$SDK_INFO") == v0.53.8 ]] || { echo 'This tooling patch targets SDK v0.53.8 only' >&2; exit 1; }
SDK_DIR=$(jq -er '.Dir' <<<"$SDK_INFO")
TMP=$(mktemp -d -t xion-testnet-build.XXXXXX)
trap 'chmod -R u+w "$TMP"; rm -rf -- "$TMP"' EXIT
# Do not modify the module cache or the checked-in dependency graph.
cp -a -- "$SDK_DIR" "$TMP/sdk"
chmod -R u+w "$TMP/sdk"
patch --batch --fuzz=0 -d "$TMP/sdk" -p1 <"$ROOT/scripts/in-place-testnet-sdk.patch"
cp -- go.mod "$TMP/rehearsal.mod"
cp -- go.sum "$TMP/rehearsal.sum"
GOWORK=off go mod edit -modfile "$TMP/rehearsal.mod" -replace "github.com/cosmos/cosmos-sdk=$TMP/sdk"
mkdir -p -- "$(dirname -- "$OUTPUT")"
GOWORK=off go build -mod=readonly -modfile "$TMP/rehearsal.mod" -o "$OUTPUT" ./cmd/xiond
printf 'Built previous-version rehearsal tooling: %s\n' "$OUTPUT"
