#!/bin/sh

# Get host directory
: ${my_dir:="$(realpath $(dirname $0))"}
: ${base_dir:="$(dirname $my_dir)"}

# Build Docker Imgae
docker build $base_dir --tag burnt/xiond:develop

# Prepare volume directory in HOME
mkdir -p ${HOME}/xiond-devnet
cp $my_dir/*json ${HOME}/xiond-devnet/
cp $my_dir/entrypoint.sh ${HOME}/xiond-devnet/

# This is provided as an example, adjust as needed
exec docker run \
  --user=root \
  --workdir=/home/xiond \
  --volume=${HOME}/xiond-devnet:/home/xiond \
  --entrypoint=/home/xiond/entrypoint.sh \
  --env="HOME=/home/xiond" \
  --publish 1317:1317 \
  --publish 9090:9090 \
  --publish 26657:26657 \
  burnt/xiond:develop cosmovisor run start