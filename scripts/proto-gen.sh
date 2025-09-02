#!/usr/bin/env bash

set -eo pipefail

if [ -n "$DEBUG" ]; then
  set -x
fi

# Get the directory of this script, used to source other scripts
: ${scripts_dir:="$(realpath $(dirname $0))"}
: ${base_dir:="$(dirname $scripts_dir)"}
: ${proto_dir:="$base_dir/proto"}
: ${client_dir:="$base_dir/client"}
: ${docs_dir:="$client_dir/docs"}

# Define dependencies
deps=$(cat <<EOF
  github.com/cosmos/cosmos-sdk
  github.com/cosmos/cosmos-proto
  github.com/cosmos/ibc-go/v10
  github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10
  github.com/CosmWasm/wasmd
EOF
)

# Install selected dependencies from go.mod
go mod download $deps

# Get dependency paths
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
  find $@ -path -prune -o -name '*.proto' -print0 \
    | xargs -0 -n1 dirname | sort -u 
}

gen_gogo() {
  local dirs=$(get_proto_dirs $proto_dir)

  for dir in $dirs; do
    for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
      if grep "option go_package" $file &> /dev/null ; then
      buf generate --output $proto_dir --template $proto_dir/buf.gen.gogo.yaml $file
      fi
    done
  done

  # move proto files to the right places
  if [ -e "$base_dir/github.com/burnt-labs/xion" ]; then
  cp -rv $base_dir/github.com/burnt-labs/xion/* $base_dir/
  rm -rf $base_dir/github.com
  fi
}

gen_swagger() {
  local dirs=$(get_proto_dirs $proto_dir $proto_paths)

  use_tmp_dir $docs_dir
  # Generate swagger for each path
  for dir in $dirs; do
    # generate swagger files (filter query files)
    query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
    [[ -n "$query_file" ]] || continue

    buf generate --template $proto_dir/buf.gen.swagger.yaml $query_file
  done
  # find ./ -type f

  # combine swagger files
  # uses nodejs package `swagger-combine`.
  # all the individual swagger files need to be configured in `config.json` for merging
  
  swagger-combine ${docs_dir}/config.yaml \
    --format "json" \
    --output ${docs_dir}/static/swagger.json \
    --includeDefinitions true \
    --continueOnConflictingPaths true

  # Generate OpenAPI spec using Swagger2Openapi
  # Install required dependencies if not already installed
  npm install --prefix ./ swagger2openapi
  npm exec -- swagger2openapi ../static/swagger.json --outfile ../static/openapi.json
}


# Parse CLI parameters
if [[ $# -eq 0 ]]; then
  gen_gogo
else
  while [[ $# -gt 0 ]]; do
    case $1 in
    --gogo)
      gen_gogo
      shift
      ;;
    --swagger)
      gen_swagger
      shift
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
    esac
  done
fi

# clean up tmp dir
#rm -rf $tmp_dir
