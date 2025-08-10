#!/bin/sh
set -eux
# create users
rm -rf .testnode/
mkdir -p .testnode
APP_FILE=./build/xiond
NODE_HOME="$PWD/.testnode"
$APP_FILE config set client chain-id xionnet-1 --home $NODE_HOME
$APP_FILE config set client keyring-backend test --home $NODE_HOME
$APP_FILE config set client output json --home $NODE_HOME

yes | $APP_FILE keys add validator --home $NODE_HOME --keyring-backend test
yes | $APP_FILE keys add creator --home $NODE_HOME --keyring-backend test
yes | $APP_FILE keys add investor --home $NODE_HOME --keyring-backend test
yes | $APP_FILE keys add funder --home $NODE_HOME --keyring-backend test --pubkey "{\"@type\":\"/cosmos.crypto.secp256k1.PubKey\",\"key\":\"AtObiFVE4s+9+RX5SP8TN9r2mxpoaT4eGj9CJfK7VRzN\"}"
VALIDATOR=$($APP_FILE keys show validator -a --home $NODE_HOME --keyring-backend test)
CREATOR=$($APP_FILE keys show creator -a --home $NODE_HOME --keyring-backend test)
INVESTOR=$($APP_FILE keys show investor -a --home $NODE_HOME --keyring-backend test)
FUNDER=$($APP_FILE keys show funder -a --home $NODE_HOME --keyring-backend test)
DENOM=uxion
# setup chain
$APP_FILE init xion --chain-id xionnet-1 --home $NODE_HOME

# modify config for development
config="$NODE_HOME/config/config.toml"
if [ "$(uname)" = "Linux" ]; then
  sed -i "s/cors_allowed_origins = \[\]/cors_allowed_origins = [\"*\"]/g" $config
  sed -i "s/\"stake\"/\"$DENOM\"/g" $NODE_HOME/config/genesis.json
else
  sed -i '' "s/cors_allowed_origins = \[\]/cors_allowed_origins = [\"*\"]/g" $config
  sed -i '' "s/\"stake\"/\"$DENOM\"/g" $NODE_HOME/config/genesis.json
fi

# modify genesis params for xionnet ease of use
# x/gov params change
# reduce voting period to 2 minutes
contents="$(jq '.app_state.gov.voting_params.voting_period = "120s"' $NODE_HOME/config/genesis.json)" && echo "${contents}" >  $NODE_HOME/config/genesis.json
# reduce minimum deposit amount to 10stake
contents="$(jq '.app_state.gov.deposit_params.min_deposit[0].amount = "10"' $NODE_HOME/config/genesis.json)" && echo "${contents}" >  $NODE_HOME/config/genesis.json
# reduce deposit period to 20seconds
contents="$(jq '.app_state.gov.deposit_params.max_deposit_period = "20s"' $NODE_HOME/config/genesis.json)" && echo "${contents}" >  $NODE_HOME/config/genesis.json

$APP_FILE genesis add-genesis-account $VALIDATOR 10000000000000000uxion --home $NODE_HOME
$APP_FILE genesis add-genesis-account $CREATOR 10000000000000000uxion --home $NODE_HOME
$APP_FILE genesis add-genesis-account $INVESTOR 10000000000000000uxion --home $NODE_HOME
$APP_FILE genesis add-genesis-account $FUNDER 10000000000000000uxion --home $NODE_HOME
$APP_FILE genesis gentx validator 10000000000uxion --chain-id xionnet-1 --keyring-backend test --home $NODE_HOME
$APP_FILE genesis collect-gentxs --home $NODE_HOME
$APP_FILE genesis validate-genesis --home $NODE_HOME
$APP_FILE start --home $NODE_HOME
