#!/bin/sh

VALIDATOR_MNEMONIC="clinic tube choose fade collect fish original recipe pumpkin fantasy enrich sunny pattern regret blouse organ april carpet guitar skin work moon fatigue hurdle"
FAUCET_MNEMONIC="decorate corn happy degree artist trouble color mountain shadow hazard canal zone hunt unfold deny glove famous area arrow cup under sadness salute item"

VALIDATOR_KEY_NAME="${VALIDATOR_KEY_NAME:-local-testnet-validator}"
FAUCET_KEY_NAME="${FAUCET_KEY_NAME:-local-testnet-faucet}"

CHAIN_ID=xion-local-testnet-1
HOME_DIRECTORY=./xion/chain-data

if [[ ! -f $HOME_DIRECTORY/data/priv_validator_state.json ]]; then
  xiond init validator --chain-id $CHAIN_ID --default-denom uxion \
    --home $HOME_DIRECTORY 2>/dev/null;

  jq --argfile bytes wasm_code_bytes.json --argfile seq wasm_seq.json '.app_state.wasm.codes = $bytes | .app_state.wasm.sequences = $seq' $HOME_DIRECTORY/config/genesis.json  > temp.json;
  mv ./temp.json $HOME_DIRECTORY/config/genesis.json;

  echo $FAUCET_MNEMONIC | xiond keys add $FAUCET_KEY_NAME --recover \
    --home $HOME_DIRECTORY \
    --keyring-backend test;

  echo $VALIDATOR_MNEMONIC | xiond keys add $VALIDATOR_KEY_NAME --recover \
    --home $HOME_DIRECTORY \
    --keyring-backend test;

  VALIDATOR_ADDRESS=$(xiond keys show $VALIDATOR_KEY_NAME -a --keyring-backend test --home $HOME_DIRECTORY);
  xiond genesis add-genesis-account $VALIDATOR_ADDRESS 100000000000uxion \
    --keyring-backend test \
    --home $HOME_DIRECTORY;

  FAUCET_ADDRESS=$(xiond keys show $FAUCET_KEY_NAME -a --keyring-backend test --home $HOME_DIRECTORY);
  xiond genesis add-genesis-account $FAUCET_ADDRESS 100000000000uxion \
    --keyring-backend test \
    --home $HOME_DIRECTORY;

  xiond genesis gentx $VALIDATOR_KEY_NAME 100000000uxion --chain-id $CHAIN_ID \
    --home $HOME_DIRECTORY \
    --keyring-backend test;

  # The output here is very verbose
  xiond genesis collect-gentxs --home $HOME_DIRECTORY 2>/dev/null;

  # Enable the API.
  sed -i '/\[api\]/,+3 s/enable = false/enable = true/' $HOME_DIRECTORY/config/app.toml;

  # Bind API to all the network interfaces.
  sed -i '/\[rpc\]/,+3 s/laddr = "tcp\:\/\/127.0.0.1:26657"/laddr = "tcp\:\/\/0.0.0.0:26657"/' $HOME_DIRECTORY/config/config.toml;

  # Expand CORS to be accessible from anywhere.
  sed -i '/\[rpc\]/,+8 s/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' $HOME_DIRECTORY/config/config.toml;
fi

xiond start --trace --home $HOME_DIRECTORY
