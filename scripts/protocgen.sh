#!/usr/bin/env bash
# Use `make protogen` to run this script

set -eo pipefail

# Get the directory of this script, used to source other scripts
scripts_dir="$(realpath $(dirname $0))"
base_dir="$(dirname $scripts_dir)"
proto_dir="$base_dir/proto"

# sets $proto_dirs
cd $proto_dir
#source $scripts_dir/protoc-common.sh
proto_dirs=$(find ./xion -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)

for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep "option go_package" $file &> /dev/null ; then
      buf generate --template $proto_dir/buf.gen.gogo.yaml $file
    fi
  done
done

cd $base_dir

# move proto files to the right places
if [ -e "$base_dir/github.com/burnt-labs/xion" ]; then
  cp -rv $base_dir/github.com/burnt-labs/xion/* $base_dir/
  rm -rf $base_dir/github.com
fi
