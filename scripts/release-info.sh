#!/bin/bash
set -Eeuo pipefail

if [ -n "${DEBUG:-}" ]; then
  set -x
fi

# Get the directory of this script, used to source other scripts
: ${scripts_dir:="$(realpath $(dirname $0))"}
: ${base_dir:="$(dirname $scripts_dir)"}
: ${release_dir:="$base_dir/release"}

# set binaries file
binaries_json=${release_dir}/binaries.json

# set ref name if not set
: ${GITHUB_REF_NAME:=$(git describe --tags)}

upgrade_name=$(echo $GITHUB_REF_NAME | cut -d. -f1)

binaries=$(
  find "$release_dir" -name 'xiond_*.tar.gz' ! -name 'xiond_*darwin_all.tar.gz' | sort 
) 

binaries_list=$(
  for file in ${binaries[@]}; do
    platform=$(basename "$file" ".tar.gz" | cut -d_ -f3- | sed -E 's/^rc[0-9]*-//g; s/_/\//g')
    checksum=$(sha256sum "$file" | awk '{ print $1 }')
    echo "\"$platform\": \"https://github.com/burnt-labs/xion/releases/download/${GITHUB_REF_NAME}/$(basename "$file")?checksum=sha256:$checksum"\"
  done
)

echo "{\"binaries\": {$(paste -s -d "," <(echo "${binaries_list[@]}"))}}" | jq . > ${binaries_json}

go mod edit -json |
  jq --rawfile binaries "$binaries_json" --arg name "$upgrade_name" --arg tag "$GITHUB_REF_NAME" '{
    name: $name,
    tag: $tag,
    recommended_version: $tag,
    language: {
      type: "go",
      version: ("v" + (.Go | split(".") | first + "." + (.[1] // "")))
    },
    binaries: ($binaries | fromjson).binaries,
    sdk: {
      type: "cosmos",
      version: (.Require[] | select(.Path == "github.com/cosmos/cosmos-sdk") | .Version)
    },
    consensus: {
      type: "cometbft",
      version: (.Require[] | select(.Path == "github.com/cometbft/cometbft") | .Version)
    },
    cosmwasm: {
      version: (.Require[] | select(.Path == "github.com/CosmWasm/wasmd") | .Version),
      enabled: (.Require[] | select(.Path == "github.com/CosmWasm/wasmd") != null)
    },
    ibc: {
      type: "go",
      version: (.Require[] | select(.Path == "github.com/cosmos/ibc-go/v8") | .Version)
    }
  }' | tee "$release_dir/version.json"
