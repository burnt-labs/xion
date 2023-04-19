VALIDATOR_MNEMONIC="clinic tube choose fade collect fish original recipe pumpkin fantasy enrich sunny pattern regret blouse organ april carpet guitar skin work moon fatigue hurdle"
FAUCET_MNEMONIC="decorate corn happy degree artist trouble color mountain shadow hazard canal zone hunt unfold deny glove famous area arrow cup under sadness salute item"

VALIDATOR_KEY_NAME="${VALIDATOR_KEY_NAME:-local-testnet-validator}"
FAUCET_KEY_NAME="${FAUCET_KEY_NAME:-local-testnet-faucet}"

key_exists() {
    local key_name="$1";
    local keys_output="$(xiond keys list --output json)";
    if [[ "$keys_output" == "No records were found in keyring" ]]; then
        return "0";
    else
        return "$(echo $keys_output | jq "map(select(.name == \"$key_name\")) | length")"
    fi
}

# Create keys locally if necessary
key_exists $VALIDATOR_KEY_NAME
if [[ "$?" == "0" ]]; then
    echo "Validator key not present, creating..."
    echo $VALIDATOR_MNEMONIC | xiond keys add $VALIDATOR_KEY_NAME --recover
fi

key_exists $FAUCET_KEY_NAME
if [[ "$?" == "0" ]]; then
    echo "Faucert key not present, creating..."
    echo $FAUCET_MNEMONIC | xiond keys add $FAUCET_KEY_NAME --recover
fi
