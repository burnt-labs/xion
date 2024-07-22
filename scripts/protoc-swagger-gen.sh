#!/usr/bin/env bash

set -eo pipefail

# change to the root folder
cd "$(dirname $(realpath "$0"))/.."

mkdir -p ./tmp-swagger-gen

# Get the path of the cosmos-sdk repo from go/pkg/mod
proto_paths=$(go list -f '{{ .Dir }}' -m \
  github.com/gogo/protobuf \
  github.com/cosmos/cosmos-sdk \
  github.com/cosmos/ibc-go/v7 \
  github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7 \
  github.com/CosmWasm/wasmd \
  github.com/cosmos/cosmos-proto \
  | sed "s/$/\/proto/"
)

proto_dirs=$(
  printf "./proto\n$proto_paths" \
  | while read path; do
    find $path -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname
  done | sort -u
)

set -x
printf "$proto_dirs" | while read dir; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))

  if [[ ! -z "$query_file" ]]; then
    #buf generate --template proto/buf.gen.swagger.yaml $query_file
    protoc -I ./proto $(printf "$proto_paths" | sed 's/^/ -I /') $query_file \
    --swagger_out ./tmp-swagger-gen \
    --swagger_opt logtostderr=true \
    --swagger_opt fqn_for_swagger_name=true \
    --swagger_opt simple_operation_ids=true
  fi
done

npm install -g swagger-combine
npx swagger-combine ./client/docs/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

# clean swagger files
rm -rf ./tmp-swagger-gen