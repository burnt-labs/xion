#!/usr/bin/env bash
# Use `make protogen` to run this script

set -eo pipefail

# Get the directory of this script, used to source other scripts
scripts_dir="$(realpath $(dirname $0))"
base_dir="$(dirname $scripts_dir)"
proto_dir="$base_dir/proto"

# sets $proto_dirs
source $scripts_dir/protoc-common.sh

for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep "option go_package" $file &> /dev/null ; then
      buf generate --template $proto_dir/buf.gen.gogo.yaml $file
    fi
  done
done

# move proto files to the right places
if [ -e "github.com/burnt-labs/xion" ]; then
  cp -rv github.com/burnt-labs/xion/* ./
  rm -rf github.com
fi
