#!/usr/bin/env bash

# Use `make proto-swagger-gen` to run this script
set -eo pipefail

# Get the directory of this script, used to source other scripts
: ${scripts_dir:="$(realpath $(dirname $0))"}
: ${base_dir:="$(dirname $scripts_dir)"}
: ${proto_dir:="$base_dir/proto"}

# sets $proto_dirs
source $scripts_dir/protoc-common.sh

# work in docs directory
cd $base_dir/client/docs

# Create a temporary directory
mkdir -p tmp-swagger-gen

# Generate swagger for each path
for dir in $proto_dirs; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
  [[ -n "$query_file" ]] || continue

  buf generate --template $proto_dir/buf.gen.swagger.yaml $query_file
done

# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
mkdir -p static
swagger-combine config.yaml \
  --format "json" \
  --output static/swagger.json \
  --includeDefinitions true \
  --continueOnConflictingPaths true

# Generate OpenAPI spec using Swagger2Openapi
# Install required dependencies if not already installed
npm install --prefix tmp-swagger-gen  swagger2openapi
npm exec --prefix tmp-swagger-gen  -- swagger2openapi static/swagger.json --outfile static/openapi.json

# clean swagger files
rm -rf tmp-swagger-gen
