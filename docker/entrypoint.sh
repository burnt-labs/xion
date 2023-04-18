#!/bin/sh

if [[ ! -f /xion/data/priv_validator_state.json ]]; then
    mv /xion/config /xion-config
    xiond init fogo --chain-id xion-local-testnet-1 --home /xion
    rm -r /xion/config
    mv /xion-config /xion/config
fi

xiond start --home /xion --trace --log_level trace
