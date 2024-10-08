#!/usr/bin/env bash
set -Eeuo pipefail

# This script is used to initialize the chain

# The following environment variables are required:
DAEMON_NAME=${DAEMON_NAME:="xiond"}
DAEMON_HOME=${DAEMON_HOME:="/home/${DAEMON_NAME}/.${DAEMON_NAME}"}
CHAIN_NAME=${CHAIN_NAME:="xion-devnet-1"}
DENOM=${DENOM:="u$(echo ${DAEMON_NAME} | head -c 4)"}
MNEMONICS_JSON=${MNEMONICS_JSON:="init/mnemonics.json"}
MASTER=$(jq -r '.mnemonics[0].name' ${MNEMONICS_JSON})
NUM_VALIDATORS=${NUM_VALIDATORS:=1}
VALIDATOR_ID=0

# load environment variables
if [ -f ${HOME}/.env ]; then
    . ${HOME}/.env
fi

# cosmos_sdk_version > v0.47 genesis commands are moved under app genesis cmd
GENESIS="genesis"
ADD_INIT_FLAGS="--default-denom=${DENOM}"

sleep_random(){
    random_number=$((RANDOM % 3000))
    sleep_time=$(bc <<< "scale=3; $random_number / 1000")
    sleep $sleep_time
}

select_num(){
    if [ ${VALIDATOR_ID} -eq 0 ]; then
        sleep_random
    elif [ ${VALIDATOR_ID} -ge ${NUM_VALIDATORS} ]; then
        VALIDATOR_ID=0
        rm -rf ${HOME}/.shared/claims
    fi
    if [ -f ${HOME}/.shared/claims/validator-${VALIDATOR_ID} ]; then
        VALIDATOR_ID=$((VALIDATOR_ID + 1 ))
        select_num
    fi
    touch ${HOME}/.shared/claims/validator-${VALIDATOR_ID}
    echo "VALIDATOR_ID=${VALIDATOR_ID}" > ${HOME}/.env
}

initialize_chain(){
    local validator="$1"
    # initialize the chain
    echo "Initializing chain ${CHAIN_NAME}..."

    ${DAEMON_NAME} init ${validator} --chain-id=${CHAIN_NAME} \
    ${ADD_INIT_FLAGS} > /dev/null 2>&1
}

    # initialize an accounts
initialize_account(){
    local validator="$1"
    echo "Initializing account ${validator}..."
    jq -r ".mnemonics | .[] |select(.name ==\"${validator}\") | .mnemonic" ${MNEMONICS_JSON} |
    ${DAEMON_NAME} keys add ${validator} --keyring-backend test --recover --output json >> ${HOME}/keys.json
    ${DAEMON_NAME} ${GENESIS} add-genesis-account ${validator} 1000000000000${DENOM} --keyring-backend test
}

initialize_all_accounts(){
    local validator
    for validator in $(jq -r ".mnemonics[:${NUM_VALIDATORS}] | .[].name" ${MNEMONICS_JSON}); do
        initialize_account ${validator}
    done
}

create_gentx(){
    local validator="$1"
    echo "Creating Gentx for ${validator}..."
    # create a gentx for the validator and add it to the genesis file
    ${DAEMON_NAME} ${GENESIS} gentx ${validator} 10000000000${DENOM} \
        --keyring-backend test \
        --chain-id=${CHAIN_NAME}
    mkdir -p ${HOME}/.shared/gentxs
    cp -a ${HOME}/.*/config/gentx/* ${HOME}/.shared/gentxs/${validator}.gentx.json

}

initialize_validator(){
    local num=$1
    local validator="$(jq -r ".mnemonics[${num}].name" ${MNEMONICS_JSON})"
    initialize_chain ${validator}
    if [ ${num} -eq 0 ]; then
        initialize_all_accounts
    else
        initialize_account ${validator}
    fi
    create_gentx ${validator}
}

initialize_genesis(){
    local validator
    # wait for all gentxs to be created
    for validator in $(jq -r ".mnemonics[:${NUM_VALIDATORS}] | .[].name" ${MNEMONICS_JSON}); do
        until [ -f ${HOME}/.shared/gentxs/${validator}.gentx.json ]; do
            echo "Waiting for ${validator}.gentx.json to be created..."
            sleep 1
        done
    done

    echo "Generating Genesis..."
    ${DAEMON_NAME} ${GENESIS} collect-gentxs \
    --gentx-dir=${HOME}/.shared/gentxs \
    > /dev/null 2>&1

    # modify the genesis.json
    sed -e "s/stake/${DENOM}/g" \
        -i ${HOME}/.*/config/genesis.json

    # modify the genesis.json
    jq --slurpfile wasm ${HOME}/init/wasm-genesis.json '.app_state.wasm |= $wasm[0].wasm' ${HOME}/.*/config/genesis.json \
    > ${HOME}/.shared/genesis.json

    # copu=y final genesis back to config
    cp -a ${HOME}/.shared/genesis.json ${DAEMON_HOME}/config/genesis.json
}

wait_for_genesis(){
    # wait for genesis.json to be created
    until [ -f ${HOME}/.shared/genesis.json ]; do
        echo "Waiting for genesis.json to be created..."
        sleep 1
    done
    cp -a ${HOME}/.shared/genesis.json ${DAEMON_HOME}/config/genesis.json
}

is_sourced() {
	# https://unix.stackexchange.com/a/215279
	[ "${#FUNCNAME[@]}" -ge 2 ] \
		&& [ "${FUNCNAME[0]}" = '_is_sourced' ] \
		&& [ "${FUNCNAME[1]}" = 'source' ]
}

init(){
    # genesis.json in shared
    if [ -f ${HOME}/.shared/genesis.json ]; then
        cp -a ${HOME}/.shared/genesis.json ${DAEMON_HOME}/config/genesis.json
    fi

    if [ ! -f ${DAEMON_HOME}/config/genesis.json ]; then
        mkdir -p ${HOME}/.shared/claims
        select_num
        initialize_validator ${VALIDATOR_ID}
        if [ ${VALIDATOR_ID} -eq 0 ]; then
            initialize_genesis
        else
            wait_for_genesis
        fi
    fi
}

init_cosmovisor(){
    export DAEMON_NAME
    export DAEMON_HOME
    export DAEMON_ALLOW_DOWNLOAD_BINARIES=true
    export DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true
    export UNSAFE_SKIP_BACKUP=true
    cosmovisor init /usr/bin/xiond
}

if ! is_sourced; then
    if grep -q cosmovisor <<<$1; then
        init_cosmovisor
    fi
	init && exec "$@"
fi
