#!/usr/bin/env bash
# Rehearse v32 against an extracted, disposable snapshot using local binaries.
set -Eeuo pipefail
umask 077

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
PREV_BIN=${PREV_BIN:-}
CUR_BIN=${CUR_BIN:-}
SNAPSHOT=${SNAPSHOT:-}
GENESIS=${GENESIS:-}
WORK_DIR=${WORK_DIR:-}
SOURCE_CHAIN=xion-mainnet-1
CHAIN=rehearsal-1
UPGRADE=v32
PORT=28657
TIMEOUT=300
NODE_PID=
NODE_LOG=
NODE_HOME=

usage() {
  cat <<EOF
Usage: $0 [options]

  --snapshot PATH    Snapshot archive (required; or set SNAPSHOT)
  --prev-bin PATH    Previous binary with in-place-testnet (required; or set PREV_BIN)
  --cur-bin PATH     Prebuilt candidate (or CUR_BIN); otherwise run make build
  --work-dir PATH    NEW extraction folder (or WORK_DIR); existing paths refused
  --source-chain ID  Snapshot network ID (default: xion-mainnet-1; xion-testnet-2
                     is refused for v32, see scripts/README.md)
  --genesis PATH     Optional source genesis (or GENESIS) instead of the initialized one
  --chain ID         Isolated rehearsal chain ID (default: rehearsal-1)
  --upgrade NAME     Upgrade handler name (default: v32)
  --port NUMBER      Loopback RPC port; P2P uses port-1 (default: 28657)
  --timeout SECONDS  Deadline per startup/upgrade phase (default: 300)
  -h, --help         Show this help

Initializes a fresh local home, installs the snapshot data/wasm, converts with
the previous binary, produces blocks, submits and votes for the upgrade, then
starts this checkout's freshly built binary on that SAME home after the halt.
The original archive is untouched. Logs, state and test keys are retained.
Set PREV_BIN and SNAPSHOT in your shell or private .envrc; flags override them.
Leave CUR_BIN unset to build this checkout. The script does not load .envrc itself.
Requires Bash, jq, GNU tar, lz4, curl, a previous xiond binary, and make/Go to build.
EOF
}
log() { printf '[rehearsal] %s\n' "$*" >&2; }
die() { log "FAIL: $*"; exit 1; }
while (($#)); do
  case $1 in
    -h|--help) usage; exit 0 ;;
    --snapshot|--prev-bin|--cur-bin|--work-dir|--source-chain|--genesis|--chain|--upgrade|--port|--timeout)
      (($# >= 2)) && [[ -n $2 ]] || die "Missing value for $1"
      case $1 in
        --snapshot) SNAPSHOT=$2 ;; --prev-bin) PREV_BIN=$2 ;; --cur-bin) CUR_BIN=$2 ;;
        --work-dir) WORK_DIR=$2 ;; --genesis) GENESIS=$2 ;; --chain) CHAIN=$2 ;;
        --source-chain) SOURCE_CHAIN=$2 ;;
        --upgrade) UPGRADE=$2 ;; --port) PORT=$2 ;; --timeout) TIMEOUT=$2 ;;
      esac
      shift 2 ;;
    *) die "Unknown argument: $1" ;;
  esac
done

stop_node() {
  [[ -n $NODE_PID ]] || return 0
  if kill -0 "$NODE_PID" 2>/dev/null; then
    kill -TERM "$NODE_PID" 2>/dev/null || :
    local end=$((SECONDS + 20))
    while kill -0 "$NODE_PID" 2>/dev/null && ((SECONDS < end)); do sleep 0.2; done
    if kill -0 "$NODE_PID" 2>/dev/null; then kill -KILL "$NODE_PID" 2>/dev/null || :; fi
  fi
  local rc=0
  wait "$NODE_PID" || rc=$?
  log "Node stopped (exit $rc); logs retained"
  NODE_PID=
}
cleanup() {
  local rc=$?
  trap - EXIT
  stop_node
  [[ -z $WORK_DIR ]] || log "Retained extraction and evidence: $WORK_DIR"
  exit "$rc"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'log "Command failed at line $LINENO (exit $?)"' ERR

# Every CLI and daemon invocation below supplies the same explicit scratch home.
prev() { "$PREV_BIN" "$@" --home "$NODE_HOME"; }
current() { "$CUR_BIN" "$@" --home "$NODE_HOME"; }
rpc() { curl --silent --show-error --fail --max-time 3 "http://127.0.0.1:$PORT/$1"; }
node_alive() {
  if ! kill -0 "$NODE_PID" 2>/dev/null || { [[ -f $NODE_LOG ]] && grep -Fq '"message":"error in proxyAppConn.FinalizeBlock"' "$NODE_LOG"; }; then
    # The scheduled halt can race the loop's first marker check. It is success
    # only in the halt phase and only with the exact plan/height evidence.
    if [[ ${1:-} == upgrade ]] && upgrade_halted; then return; fi
    if [[ -f $NODE_LOG ]]; then
      jq -Rr 'fromjson? | select(.message=="error in proxyAppConn.FinalizeBlock") | .err' "$NODE_LOG" >&2
    fi
    die "Node failed; inspect $NODE_LOG (no upgrade guard is bypassed)"
  fi
}
start_node() {
  local wrapper=$1 logfile=$2; shift 2
  NODE_LOG=$logfile
  log "Starting $wrapper $* (home $NODE_HOME); log: $logfile"
  # Launch the binary itself, not an asynchronously forked wrapper shell, so
  # NODE_PID always belongs to the daemon we must terminate and reap.
  local binary=$PREV_BIN
  [[ $wrapper != current ]] || binary=$CUR_BIN
  "$binary" "$@" --home "$NODE_HOME" "${NODE_FLAGS[@]}" >"$logfile" 2>&1 &
  NODE_PID=$!
}
wait_blocks() {
  local target=$1 expected_chain=${2:-$CHAIN} end=$((SECONDS + TIMEOUT)) height header
  while ((SECONDS < end)); do
    node_alive
    if height=$(rpc status 2>/dev/null | jq -er '.result.sync_info.latest_block_height | tonumber') && ((height >= target)); then
      header=$(rpc "block?height=$height" | jq -er '.result.block.header.chain_id')
      [[ $header == "$expected_chain" ]] || die "Committed chain ID $header is not $expected_chain"
      printf '%s\n' "$height"
      return
    fi
    sleep 1
  done
  die "Timed out waiting for $expected_chain height $target; inspect logs"
}
query_versions() {
  "$1" query upgrade module-versions --node "$RPC_URL" -o json | jq -e '[.module_versions[] | {key:.name,value:((.version // 0)|tonumber)}] | from_entries'
}

normalize_params() {
  local selector=.params
  [[ $1 != globalfee ]] || selector=.
  jq -eS "$selector | select(type==\"object\")"
}

upgrade_halted() {
  [[ -s $NODE_HOME/data/upgrade-info.json ]] &&
    jq -e --arg name "$UPGRADE" --argjson height "$UPGRADE_HEIGHT" '.name==$name and (.height|tonumber)==$height' \
      "$NODE_HOME/data/upgrade-info.json" >/dev/null &&
    grep -Fq "NEEDED at height: $UPGRADE_HEIGHT" "$WORK_DIR/in-place.log"
}

submit_tx() {
  local output=$1; shift
  local response hash tx end=$((SECONDS + TIMEOUT))
  response=$(prev tx "$@" --from rehearsal-operator --keyring-backend test --chain-id "$CHAIN" \
    --node "$RPC_URL" --gas 5000000 --fees "5000000$BOND_DENOM" --yes -o json)
  jq -e '(.code // 0)==0' <<<"$response" >/dev/null || die "CheckTx failed: $response"
  hash=$(jq -er '.txhash | select(length>0)' <<<"$response")
  while ((SECONDS < end)); do
    node_alive
    if tx=$(rpc "tx?hash=0x$hash" 2>/dev/null); then
      if jq -e 'has("result")' <<<"$tx" >/dev/null; then
        jq -e '(.result.tx_result.code // 0)==0' <<<"$tx" >/dev/null || die "DeliverTx failed: $tx"
        printf '%s\n' "$tx" >"$output"
        return
      fi
      jq -e '.error.data | contains("not found")' <<<"$tx" >/dev/null || die "Transaction lookup failed: $tx"
    fi
    sleep 1
  done
  die "Transaction $hash was not committed"
}

# Validate the complete archive before extraction; do not import links or devices.
extract_snapshot() {
  local archive=$1 destination=$2 listing permissions owner size date time member
  local -a databases=()
  [[ ! -e $destination && ! -L $destination ]] || die "Extraction exists: $destination"
  mkdir -m 700 -- "$destination"
  listing="$destination/.snapshot-members"
  log "Validating archive members: $archive"
  if ! lz4 -dc -- "$archive" |
    LC_ALL=C TAR_OPTIONS= tar --list --verbose --numeric-owner --full-time \
      --absolute-names --quoting-style=escape --file=- >"$listing"; then
    die 'Archive validation failed (lz4/tar); partial output retained'
  fi
  while IFS=' ' read -r permissions owner size date time member; do
    [[ -n $permissions && -n $member ]] || die 'Unparseable archive member'
    case ${permissions:0:1} in -|d) ;; *) die "Refusing link/device/special member: $member" ;; esac
    case $member in /*|..|../*|*/../*|*/..) die "Refusing unsafe archive path: $member" ;; esac
  done <"$listing"
  rm -- "$listing"
  log "Extracting archive into $destination"
  if ! lz4 -dc -- "$archive" |
    TAR_OPTIONS= tar --extract --file=- --directory="$destination" \
      --no-same-owner --no-same-permissions --delay-directory-restore; then
    die 'Archive extraction failed (lz4/tar); partial output retained'
  fi
  find "$destination" -type d -exec chmod 700 -- {} +
  find "$destination" -type f -exec chmod 600 -- {} +
  mapfile -d '' -t databases < <(find "$destination" -type d -path '*/data/application.db' -print0)
  ((${#databases[@]} == 1)) || die 'Expected exactly one extracted data/application.db directory'
  printf '%s\n' "${databases[0]%/data/application.db}"
}

[[ -n $PREV_BIN ]] || die 'Set PREV_BIN or pass --prev-bin PATH'
[[ -n $SNAPSHOT && -f $SNAPSHOT ]] || die 'Set SNAPSHOT or pass --snapshot PATH to an existing archive'
for tool in jq tar lz4 curl realpath sha256sum; do command -v "$tool" >/dev/null || die "Missing command: $tool"; done
[[ $PORT =~ ^[0-9]+$ && $TIMEOUT =~ ^[0-9]+$ ]] || die 'Port and timeout must be positive integers'
((PORT > 1025 && PORT < 65536 && TIMEOUT > 0)) || die 'Invalid port or timeout'
# The v32 upgrade backfills checksum-derived abstract accounts only on the chain
# IDs listed in app/upgrades.go (checksumAccountCodes), keyed by the chain ID
# the upgrade runs under. A rehearsal runs under its own chain ID, so a
# xion-testnet-2 snapshot would skip that backfill and still pass.
if [[ $UPGRADE == v32 && $SOURCE_CHAIN == xion-testnet-2 ]]; then
  die 'xion-testnet-2 cannot be rehearsed for v32: its checksum-account backfill is keyed by chain ID and would be skipped under the rehearsal chain ID'
fi
PREV_BIN=$(realpath -e -- "$PREV_BIN")
SNAPSHOT=$(realpath -e -- "$SNAPSHOT")
[[ -x $PREV_BIN ]] || die 'Supply an executable previous binary'
[[ -z $GENESIS ]] || GENESIS=$(realpath -e -- "$GENESIS")
if [[ -z $WORK_DIR ]]; then WORK_DIR="${SNAPSHOT%.tar.lz4}.rehearsal-$(date +%Y%m%d-%H%M%S)-$$"; fi
WORK_DIR=$(realpath -m -- "$WORK_DIR")
[[ ! -e $WORK_DIR && ! -L $WORK_DIR ]] || die "Extraction folder already exists: $WORK_DIR"
# Fail before extraction if anything listens on the RPC or P2P port. A
# conversion that cannot bind them dies after it has rewritten consensus state.
# The node binds both on 127.0.0.1, so a successful loopback connection means
# the bind would fail.
port_in_use() { (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null; }
for port in "$PORT" "$((PORT - 1))"; do
  if port_in_use "$port"; then die "Port $port is in use (RPC uses --port, P2P uses --port minus 1)"; fi
done
if [[ -z $CUR_BIN ]]; then
  command -v make >/dev/null || die 'Missing command: make'
  CUR_BIN=$(realpath -m -- "$ROOT/build/xiond")
  [[ $PREV_BIN != "$CUR_BIN" ]] || die 'Previous binary must not be the candidate build output'
  log "Building candidate from $ROOT with make build"
  make -C "$ROOT" build >&2
fi
CUR_BIN=$(realpath -e -- "$CUR_BIN")
[[ -x $CUR_BIN && $PREV_BIN != "$CUR_BIN" ]] || die 'Supply distinct executable previous/current binaries'
# Initialize BEFORE unpacking, as Polkachu's data/wasm archives omit configuration.
# No archived validator credentials or network configuration are installed.
mkdir -m 700 -- "$WORK_DIR"
NODE_HOME="$WORK_DIR/node"
prev init rehearsal --chain-id "$SOURCE_CHAIN" >"$WORK_DIR/init.log" 2>&1
prev in-place-testnet --help >"$WORK_DIR/previous-command-help.txt"
grep -q -- '--trigger-testnet-upgrade' "$WORK_DIR/previous-command-help.txt" || die 'Previous binary lacks in-place-testnet upgrade tooling'
prev keys add rehearsal-operator --keyring-backend test --output json >"$WORK_DIR/operator.json" 2>"$WORK_DIR/operator.log"
SNAPSHOT_HOME=$(extract_snapshot "$SNAPSHOT" "$WORK_DIR/snapshot")
[[ -n $GENESIS ]] || GENESIS="$NODE_HOME/config/genesis.json"
[[ $(jq -er '.chain_id' "$GENESIS") == "$SOURCE_CHAIN" ]] || die 'Source genesis chain ID disagrees with --source-chain'
[[ $SOURCE_CHAIN != "$CHAIN" ]] || die 'Rehearsal chain ID must differ from the source network'
cp -- "$GENESIS" "$WORK_DIR/source-genesis.json"
cp -- "$WORK_DIR/source-genesis.json" "$NODE_HOME/config/genesis.json"
mv -- "$NODE_HOME/data" "$NODE_HOME/data.initial"
mv -- "$SNAPSHOT_HOME/data" "$NODE_HOME/data"
[[ -d $SNAPSHOT_HOME/wasm ]] || die 'Snapshot must include wasm/ alongside data/'
if [[ -d $NODE_HOME/wasm ]]; then mv -- "$NODE_HOME/wasm" "$NODE_HOME/wasm.initial"; fi
mv -- "$SNAPSHOT_HOME/wasm" "$NODE_HOME/wasm"
sha256sum -- "$PREV_BIN" "$CUR_BIN" >"$WORK_DIR/binaries.sha256"
log "All xiond commands use extracted home: $NODE_HOME"
OPERATOR=$(jq -er '.address' "$WORK_DIR/operator.json")
# The source archive may contain an old upgrade-info file unrelated to this run.
if [[ -f $NODE_HOME/data/upgrade-info.json ]]; then mv -- "$NODE_HOME/data/upgrade-info.json" "$WORK_DIR/source-upgrade-info.json"; fi
RPC_URL="http://127.0.0.1:$PORT"
NODE_FLAGS=(--rpc.laddr "tcp://127.0.0.1:$PORT" --p2p.laddr "tcp://127.0.0.1:$((PORT - 1))"
  --p2p.pex=false --p2p.seeds= --p2p.persistent_peers= --grpc.enable=false --api.enable=false
  --consensus.timeout_commit 1s --minimum-gas-prices 0uxion --log_format json)
# Query the isolated source state before conversion. This new local validator key
# is not in the source validator set; with no peers it cannot advance that chain.
start_node prev "$WORK_DIR/source.log" start
SOURCE_HEIGHT=$(wait_blocks 1 "$SOURCE_CHAIN")
query_versions prev >"$WORK_DIR/before-versions.json"
for module in globalfee jwk; do
  prev query "$module" params --node "$RPC_URL" -o json | normalize_params "$module" >"$WORK_DIR/before-$module.json"
done
stop_node
log "Source snapshot height: $SOURCE_HEIGHT"
start_node prev "$WORK_DIR/in-place.log" in-place-testnet "$CHAIN" "$OPERATOR" --skip-confirmation
CONVERTED_HEIGHT=$(wait_blocks "$((SOURCE_HEIGHT + 3))")
query_versions prev >"$WORK_DIR/converted-versions.json"
cmp -- "$WORK_DIR/before-versions.json" "$WORK_DIR/converted-versions.json" || die 'Conversion changed module versions'
prev query staking params --node "$RPC_URL" -o json >"$WORK_DIR/source-staking.json"
prev query gov params --node "$RPC_URL" -o json >"$WORK_DIR/source-gov.json"
prev query auth module-account gov --node "$RPC_URL" -o json >"$WORK_DIR/governance-account.json"
BOND_DENOM=$(jq -er '.params.bond_denom' "$WORK_DIR/source-staking.json")
AUTHORITY=$(jq -er '.account.value.address' "$WORK_DIR/governance-account.json")
DEPOSIT=$(jq -er '[.params.min_deposit[] | .amount+.denom] | join(",") | select(length>0)' "$WORK_DIR/source-gov.json")
UPGRADE_HEIGHT=$((CONVERTED_HEIGHT + 60))
jq -n --arg authority "$AUTHORITY" --arg upgrade "$UPGRADE" --arg height "$UPGRADE_HEIGHT" --arg deposit "$DEPOSIT" \
  '{messages:[{"@type":"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",authority:$authority,plan:{name:$upgrade,height:$height,info:"Local snapshot rehearsal"}}],
    metadata:"",deposit:$deposit,title:"Local upgrade rehearsal",summary:"Upgrade the isolated snapshot clone",expedited:false}' >"$WORK_DIR/proposal.json"
submit_tx "$WORK_DIR/proposal-tx.json" gov submit-proposal "$WORK_DIR/proposal.json"
PROPOSAL_ID=$(jq -er '.result.tx_result.events[] | select(.type=="submit_proposal") | .attributes[] | select(.key=="proposal_id") | .value' "$WORK_DIR/proposal-tx.json")
[[ $PROPOSAL_ID =~ ^[0-9]+$ ]] || die "Invalid proposal ID: $PROPOSAL_ID"
submit_tx "$WORK_DIR/vote-tx.json" gov vote "$PROPOSAL_ID" yes
end=$((SECONDS + TIMEOUT))
PASSED=false
while ((SECONDS < end)); do
  node_alive
  prev query gov proposal "$PROPOSAL_ID" --node "$RPC_URL" -o json >"$WORK_DIR/proposal-status.json"
  status=$(jq -er '.proposal.status' "$WORK_DIR/proposal-status.json")
  case $status in
    PROPOSAL_STATUS_PASSED) PASSED=true; break ;;
    PROPOSAL_STATUS_REJECTED|PROPOSAL_STATUS_FAILED) die "Proposal did not pass: $status" ;;
  esac
  sleep 1
done
[[ $PASSED == true ]] || die 'Timed out waiting for the upgrade proposal to pass'
prev query upgrade plan --node "$RPC_URL" -o json >"$WORK_DIR/plan.json"
jq -e --arg name "$UPGRADE" --argjson height "$UPGRADE_HEIGHT" '.plan.name==$name and (.plan.height|tonumber)==$height' \
  "$WORK_DIR/plan.json" >/dev/null || die 'Passed proposal scheduled a different upgrade'
log "Proposal $PROPOSAL_ID passed; $UPGRADE scheduled at $UPGRADE_HEIGHT"
end=$((SECONDS + TIMEOUT))
while ((SECONDS < end)); do
  if upgrade_halted; then break; fi
  node_alive upgrade
  sleep 1
done
upgrade_halted || die 'Previous binary did not reach the expected governance-scheduled upgrade halt'
cp -- "$NODE_HOME/data/upgrade-info.json" "$WORK_DIR/upgrade-info.json"
stop_node
log "Previous binary reached $UPGRADE at height $UPGRADE_HEIGHT; handing SAME home to current binary"
start_node current "$WORK_DIR/post-upgrade.log" start
FINAL_HEIGHT=$(wait_blocks "$((UPGRADE_HEIGHT + 3))")
query_versions current >"$WORK_DIR/after-versions.json"
current query upgrade applied "$UPGRADE" --node "$RPC_URL" -o json >"$WORK_DIR/applied.json"
jq -e --argjson height "$UPGRADE_HEIGHT" '(.height|tonumber)==$height' "$WORK_DIR/applied.json" >/dev/null || die 'Upgrade was not applied at the expected height'
# These expectations are the v32 release contract, not mere successful queries.
if [[ $UPGRADE == v32 ]]; then
  jq -e '.auth==7 and .staking==6 and .mint==3 and .globalfee==3 and .jwk==4 and .abstractaccount==3 and .zk==3 and .dkim==2' \
    "$WORK_DIR/after-versions.json" >/dev/null || die 'Unexpected v32 module versions'
  # Historical version-map entries can remain after a module is removed.
  # Prove the actual stores are unmounted rather than treating that history
  # as an active module registry.
  for store in group nft circuit crisis; do
    rpc "abci_query?path=%22/store/$store/key%22&data=0x00" >"$WORK_DIR/removed-store-$store.json"
    jq -e --arg store "$store" '(.result.response.code // 0)!=0 and (.result.response.log | contains("no such store: "+$store))' \
      "$WORK_DIR/removed-store-$store.json" >/dev/null || die "Retired store $store remains queryable or returned an unexpected error"
  done
fi
for module in auth staking mint globalfee jwk wasm abstract-account; do
  current query "$module" params --node "$RPC_URL" -o json >"$WORK_DIR/params-$module.json"
done
for module in globalfee jwk; do
  normalize_params "$module" <"$WORK_DIR/params-$module.json" >"$WORK_DIR/after-$module.json"
  cmp -- "$WORK_DIR/before-$module.json" "$WORK_DIR/after-$module.json" || die "$module params changed across upgrade"
done
jq -n --arg chain "$CHAIN" --arg home "$NODE_HOME" --arg upgrade "$UPGRADE" --argjson height "$UPGRADE_HEIGHT" --argjson final "$FINAL_HEIGHT" \
  '{chain_id:$chain,home:$home,upgrade:$upgrade,upgrade_height:$height,post_upgrade_height:$final}' >"$WORK_DIR/result.json"
stop_node
log "PASS: $UPGRADE applied at $UPGRADE_HEIGHT; blocks resumed through $FINAL_HEIGHT"
