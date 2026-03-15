#!/bin/sh
set -eux

rm -rf .testnode/
mkdir -p .testnode
APPD=./build/xiond
XIOND_HOME="$PWD/.testnode"
$APPD config set client chain-id localnet-1 --home $XIOND_HOME
$APPD config set client keyring-backend test --home $XIOND_HOME
$APPD config set client output json --home $XIOND_HOME

yes | $APPD keys add validator --home $XIOND_HOME --keyring-backend test
VALIDATOR=$($APPD keys show validator -a --home $XIOND_HOME --keyring-backend test)
DENOM=stake


$APPD init xion --chain-id localnet-1 --home $XIOND_HOME
$APPD config set app minimum-gas-prices 0.0025$DENOM --home $XIOND_HOME

$APPD genesis add-genesis-account $VALIDATOR "1000000000000000$DENOM" --home $XIOND_HOME --keyring-backend test
$APPD genesis gentx validator "1000000000$DENOM" --chain-id localnet-1 --keyring-backend test --home $XIOND_HOME 
$APPD genesis collect-gentxs --home $XIOND_HOME
$APPD genesis validate-genesis --home $XIOND_HOME

# set db backend to rocksdb
sed -i 's/^db_backend = .*/db_backend = "rocksdb"/' "$XIOND_HOME/config/config.toml"
sed -i 's/^app-db-backend = .*/app-db-backend = "rocksdb"/' "$XIOND_HOME/config/app.toml"
sed -i 's/^timeout_propose = .*/timeout_propose = "100ms"/' "$XIOND_HOME/config/config.toml"
sed -i 's/^timeout_commit = .*/timeout_commit = "100ms"/' "$XIOND_HOME/config/config.toml"

$APPD start --home $XIOND_HOME
