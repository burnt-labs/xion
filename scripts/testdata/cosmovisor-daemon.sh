#!/bin/sh
# Deterministic stand-in for xiond in the cosmovisor launcher tests
# (scripts/cosmovisor_image_test.go). It records what cosmovisor did to it
# under $DAEMON_HOME/fixture; the tests assert cosmovisor's behaviour, not this
# script's. Not a substitute for the real-state test, which runs real xiond.
set -eu

self=$(readlink -f "$0")
case "$self" in
    */cosmovisor/genesis/bin/*) id=genesis ;;
    */cosmovisor/upgrades/*/bin/*) id=$(basename "$(dirname "$(dirname "$self")")") ;;
    *) id=unknown ;;
esac
out="${DAEMON_HOME:?}/fixture"
mkdir -p "$out"

case "${1:-}" in
    version)
        echo "fixture-daemon $id"
        ;;
    status)
        printf '{"sync_info":{"latest_block_height":"10"}}\n'
        ;;
    pre-upgrade)
        echo "$id" >>"$out/pre-upgrade"
        ;;
    start)
        printf '%s\n' "$@" >"$out/start-$id.args"
        trap 'touch "$out/term-$id"; [ -n "${FIXTURE_IGNORE_TERM:-}" ] || exit 0' TERM
        trap 'touch "$out/int-$id"; exit 0' INT
        while :; do
            sleep 0.1
        done
        ;;
    *)
        echo "fixture-daemon: unsupported command: $*" >&2
        exit 2
        ;;
esac
