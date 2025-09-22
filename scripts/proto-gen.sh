#!/bin/sh

# Proto generation script for Xion
# 
# This script can be used in two ways:
# 1. Run directly: ./proto-gen.sh [--gogo|--swagger]
# 2. Source and use functions: 
#    source proto-gen.sh
#    gen_gogo        # Generate gogo protobuf files
#    gen_swagger     # Generate swagger documentation
#
# Available functions when sourced:
# - gen_gogo: Generate gogo protobuf files
# - gen_swagger: Generate swagger documentation
# - get_proto_dirs: Find all subdirectories with .proto files
# - use_tmp_dir: Create and use a temporary directory
# - show_help: Display usage information
# - main: Main CLI handler

set -eo pipefail

if [ -n "$DEBUG" ]; then
  set -x
fi

# Get the directory of this script, used to source other scripts
scripts_dir="$(cd "$(dirname "$0")" && pwd)"
base_dir="$(dirname "$scripts_dir")"
proto_dir="$base_dir/proto"
client_dir="$base_dir/client"
docs_dir="$client_dir/docs"

# Define dependencies
<<<<<<< Updated upstream
deps=$(
  cat <<EOF
  github.com/cosmos/cosmos-sdk
  github.com/cosmos/cosmos-proto
  github.com/cosmos/ibc-go/v8
  github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8
  github.com/osmosis-labs/fee-abstraction/v8
  github.com/CosmWasm/wasmd
EOF
)
=======
deps="github.com/cosmos/cosmos-sdk
github.com/cosmos/cosmos-proto
github.com/cosmos/ibc-go/v10
github.com/CosmWasm/wasmd
github.com/gogo/protobuf
github.com/burnt-labs/abstract-account
cosmossdk.io/x/evidence
cosmossdk.io/x/feegrant
cosmossdk.io/x/nft
cosmossdk.io/x/upgrade
github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10
github.com/strangelove-ventures/tokenfactory
"
>>>>>>> Stashed changes

# Install selected dependencies from go.mod
echo "installing dependencies"
(cd ${base_dir} && go mod download)

# Get dependency paths
echo "getting paths for $deps"
proto_paths=$(go list -f '{{ .Dir }}' -m $deps | sed "s/$/\/proto/")

use_tmp_dir() {
  local path="$1"
  if [ -n "$path" ]; then
    mkdir -p $path
    tmp_dir=$(mktemp -d -p $path -t tmp-XXXXXX)
  else
    tmp_dir=$(mktemp -d -t tmp-XXXXXX)
  fi
  trap 'rm -rf -- "$tmp_dir"' EXIT
  cd $tmp_dir
}

get_proto_dirs() {
  # Find all subdirectories with .proto files
<<<<<<< Updated upstream
  find $@ -path -prune -o -name '*.proto' -print0 |
    xargs -0 -n1 dirname | sort -u
=======
  find "$@" -name '*.proto' -print0 2>/dev/null | \
    xargs -0 -n1 dirname 2>/dev/null | \
    sort -u 2>/dev/null || true
>>>>>>> Stashed changes
}

gen_gogo() {
  local dirs=$(get_proto_dirs $proto_dir)
<<<<<<< Updated upstream

  for dir in $dirs; do
    for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
      if grep "option go_package" $file &>/dev/null; then
        buf generate --output $proto_dir --template $proto_dir/buf.gen.gogo.yaml $file
      fi
    done
  done

  # move proto files to the right places
  if [ -e "$base_dir/github.com/burnt-labs/xion" ]; then
    cp -rv $base_dir/github.com/burnt-labs/xion/* $base_dir/
    rm -rf $base_dir/github.com
=======
  buf generate --output "$proto_dir" --template "$proto_dir/buf.gen.gogo.yaml" "$proto_dir"
  # move proto files to the right places
  if [ -e "$base_dir/github.com/burnt-labs/xion" ]; then
    cp -rv "$base_dir/github.com/burnt-labs/xion/"* "$base_dir/"
    rm -rf "$base_dir/github.com"
>>>>>>> Stashed changes
  fi
}

gen_pulsar() {
  go install cosmossdk.io/orm/cmd/protoc-gen-go-cosmos-orm@latest #2>/dev/null

  local dirs=$(get_proto_dirs $proto_dir)
  buf generate --output "${base_dir}/api" --template "$proto_dir/buf.gen.pulsar.yaml" "$proto_dir"
  
  # remove problematic generated files
  rm $base_dir/api/xion/feeabs/v1beta1/osmosisibc.pulsar.go
}

gen_swagger() {
  local dirs=$(get_proto_dirs "$proto_dir" $proto_paths)

  use_tmp_dir "$docs_dir"
  # Generate swagger for each path
  for dir in $dirs; do
    # generate swagger files (filter query files)
    query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
    [ -n "$query_file" ] || continue
    
    # Skip problematic dependencies that have incompatible imports
    if echo "$query_file" | grep -q "tokenfactory"; then
      continue
    fi

    buf generate --template "$proto_dir/buf.gen.docs.yaml" "$query_file"
  done

  # combine swagger files
  # uses nodejs package `swagger-combine`.
  # all the individual swagger files need to be configured in `config.json` for merging
  
  swagger-combine "${docs_dir}/config.yaml" \
    --format "json" \
    --output "${docs_dir}/static/swagger.json" \
    --includeDefinitions true \
    --continueOnConflictingPaths true

  # Generate OpenAPI spec using Swagger2Openapi
  # Install required dependencies if not already installed
  npm install --prefix ./ swagger2openapi
  npm exec -- swagger2openapi ../static/swagger.json --outfile ../static/openapi.json
}

# Show help message
show_help() {
  echo "Usage: $0 [--gogo|--openapi|--swagger|--help]"
  echo "  --gogo     Generate gogo protobuf files (default)"
  echo "  --openapi  Generate OpenAPI documentation"
  echo "  --swagger  Generate OpenAPI documentation (alias for --openapi)"
  echo "  --help     Show this help message"
}

# Main function to handle CLI parameters
main() {
  if [ $# -eq 0 ]; then
    gen_gogo
    exit 0
  fi
  while [ $# -gt 0 ]; do
    case $1 in
      --gogo)
        gen_gogo
        shift
        ;;
      --openapi|--swagger)
        gen_swagger
        shift
        ;;
      --pulsar)
        gen_pulsar
        shift
        ;;
      --help|-h)
        show_help
        return 0
        ;;
      *)
        echo "Error: Unknown option '$1'" >&2
        show_help
        return 1
        ;;
    esac
  done
}

# Only execute main if script is run directly (not sourced)
# This works in all POSIX shells including ash, bash, zsh, dash
if [ "${0##*/}" = "proto-gen.sh" ]; then
  main "$@"
fi

