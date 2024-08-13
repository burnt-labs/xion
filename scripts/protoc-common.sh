#!/usr/bin/env bash

# this script is sourced from protocgen.sh and protoc-swagger-gen.sh
# it sets the proto_dirs variable used in each

set -eo pipefail

# Get the directory of this script, used to source other scripts
: ${scripts_dir:="$(realpath $(dirname $0))"}
: ${base_dir:="$(dirname $scripts_dir)"}
: ${proto_dir:="$base_dir/proto"}

# Define dependencies
# deps=$(cat <<EOF
#   github.com/cosmos/cosmos-sdk
#   github.com/cosmos/ibc-go/v8
#   github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8
#   github.com/CosmWasm/wasmd
# EOF
# )
deps=$(
  egrep '^\s*github.com/(cosmos/(cosmos-sdk|ibc-go\/v|ibc-apps\/middle)|CosmWasm/wasmd)' \
  $base_dir/go.mod | cut -d ' ' -f 1
)

# Install dependencies
go mod download $deps

# Get dependency paths
proto_paths=$(go list -f '{{ .Dir }}' -m $deps | sed "s/$/\/proto/")

# Find all subdirectories with .proto files
proto_dirs=$(find $proto_dir $proto_paths -path -prune -o -name '*.proto' -print0 \
  | xargs -0 -n1 dirname | sort -u
)