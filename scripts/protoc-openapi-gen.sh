#!/usr/bin/env bash

# this script is called from protoc-swagger-gen.sh
set -eo pipefail

echo "Generating Protobuf Openapi"

# Get the directory of this script, used to source other scripts
: ${scripts_dir:="$(realpath $(dirname $0))"}
: ${base_dir:="$(dirname $scripts_dir)"}
: ${proto_dir:="$base_dir/proto"}

# work in docs directory
cd $base_dir/client/docs

# Create the directory
mkdir -p static/openapi

# Generate OpenAPI spec using Swagger2Openapi
# Install required dependencies if not already installed
npm install swagger2openapi @redocly/cli
npx swagger2openapi static/swagger/swagger.yaml --yaml --outfile static/openapi/openapi.yaml


# Generate documentation using ReDoc
# Install ReDoc if not already installed
# @redocly/cli
# npx @redocly/cli build-docs static/openapi/openapi.yaml --output static/openapi/index.html
