#!/bin/bash
set -Eeuo pipefail

if [ -n "${DEBUG:-}" ]; then
  set -x
fi

# Get the directory of this script, used to source other scripts
: ${scripts_dir:="$(realpath $(dirname $0))"}
: ${base_dir:="$(dirname $scripts_dir)"}
: ${release_dir:="$base_dir/release"}

# set ref name if not set
: ${GITHUB_REF_NAME:=$(git describe --tags)}

binaries=$(
  find "$release_dir" -name 'xiond_*.zip' ! -name 'xiond_*darwin_all.zip'
) 

binaries_list=$(
  for file in ${binaries[@]}; do
    platform=$(basename "$file" ".zip" | cut -d_ -f3- | sed -E 's/^rc[0-9]*-//g; s/_/\//g')
    checksum=$(sha256sum "$file" | awk '{ print $1 }')
    echo "\"$platform\": \"https://github.com/burnt-labs/xion/releases/download/${GITHUB_REF_NAME}/$(basename "$file")?checksum=sha256:$checksum"\"
  done
)

binaries_json=$(echo "{\"binaries\": {$(paste -s -d "," <(echo "${binaries_list[@]}"))}}" | jq -c .)

upgrade_name=$(echo $GITHUB_REF_NAME | cut -d. -f1)
  
go mod edit -json | 
  jq --argjson binaries "$binaries_json" --arg name $upgrade_name --arg tag $GITHUB_REF_NAME '{
    name: $name,
    tag: $tag,
    go_version: .Go,
    cosmos_sdk_version: (.Require[] | select(.Path == "github.com/cosmos/cosmos-sdk") | .Version),
    cosmwasm_enabled: (.Require[] | select(.Path == "github.com/CosmWasm/wasmd") != null),
    cosmwasm_version: (.Require[] | select(.Path == "github.com/CosmWasm/wasmd") | .Version),
    ibc_go_version: (.Require[] | select(.Path == "github.com/cosmos/ibc-go/v8") | .Version),
    consensus: {
      type: "cometbft",
      version: (.Require[] | select(.Path == "github.com/cometbft/cometbft") | .Version)
    },
    binaries: $binaries.binaries
}' | tee "$release_dir/version.json"
