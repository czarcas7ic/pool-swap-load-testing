#!/bin/bash
GAMM_POOL_TYPE="/osmosis.gamm.v1beta1.Pool"
CL_POOL_TYPE="/osmosis.concentratedliquidity.v1beta1.Pool"
CW_POOL_TYPE="/osmosis.cosmwasmpool.v1beta1.CosmWasmPool"
CHAIN_ID="localosmosis"
KEYRING_NAME="lo-test2"
KEYRING_BACKEND="test"
BROADCAST_MODE="sync"
FEES="100000uosmo"
GAS=750000
SEQUENCE=1
TX_FLAGS="--from=$KEYRING_NAME --keyring-backend=$KEYRING_BACKEND -b=$BROADCAST_MODE --chain-id=$CHAIN_ID --fees=$FEES --gas=$GAS -y -o=json"

# OSMO_GAMM_POOL_IDS=(1 2 3 5 7 9 11 15 23 24 25 27 30 31 32 33 34 35 36 39 40 41 42 43 44 45 47 48 49 50 51 52 53 54 56 57 58 60 61 65) # 40 pools
# OSMO_CL_POOL_IDS=(1066 1076 1077 1088 1089 1090 1091 1092 1093 1094 1095 1096 1097 1098 1099 1100 1101 1102 1103 1104 1105 1106 1107 1108 1109 1110 1111 1112 1113 1114 1115 1133 1134 1135 1145) # 40 pools
# OSMO_CW_POOL_IDS=(1463 1572 1575 1579 1584 1616 1635 1642 1643 1692) # 10 pools

OSMO_GAMM_POOL_IDS=(1) # 40 pools
OSMO_CL_POOL_IDS=(1066) # 40 pools
OSMO_CW_POOL_IDS=(1463) # 10 pools
POOL_IDS=("${OSMO_GAMM_POOL_IDS[@]}" "${OSMO_CL_POOL_IDS[@]}" "${OSMO_CW_POOL_IDS[@]}")

QUERY_MSG='{
  "pool": {}
}'

QUERY_MSG_2='{
  "get_total_pool_liquidity": {}
}'

retrieve_status() {
    status=$(osmosisd status 2>&1)
    latest_block_height=$(echo "$status" | jq -r '.sync_info.latest_block_height // 0')
    echo $latest_block_height
}

set_sequence() {
    sequence=$(osmosisd query auth account $(osmosisd keys show $KEYRING_NAME -a) --output json | jq -r '.account.value.sequence' 2>&1)
    echo "sequence: $sequence" >&2
    SEQUENCE=$sequence
    echo "Sequence: $SEQUENCE" >&2
}

wait_for_next_block() {
    initial_height=0
    while [ $initial_height -eq 0 ]; do
        initial_height=$(retrieve_status)
        echo "Initial height: $initial_height" >&2
        sleep 0.05
    done

    target_height=$((initial_height + 1))

    current_height=$initial_height
    while [ $current_height -lt $target_height ]; do
        current_height=$(retrieve_status)
        sleep 0.05
    done

    set_sequence
}

get_pool() {
    response=$(osmosisd q poolmanager pool $1 --output json)
    pool_type=$(echo $response | jq -r '.pool["@type"]')
    echo "$pool_type" "$response"
}

get_non_osmo_pool_asset() {
    output=$(get_pool $1)
    pool_type=$(echo "$output" | awk '{print $1}')
    response=$(echo "$output" | cut -d' ' -f2-)

    if [ "$pool_type" = "$GAMM_POOL_TYPE" ]; then
        denom=$(echo "$response" | jq -r '.pool.pool_assets[] | select(.token.denom != "uosmo") | .token.denom' | head -n 1)
        echo $denom
    elif [ "$pool_type" = "$CL_POOL_TYPE" ]; then
        token0=$(echo "$response" | jq -r '.pool.token0')
        token1=$(echo "$response" | jq -r '.pool.token1')
        if [ "$token0" != "uosmo" ]; then
            echo $token0
        elif [ "$token1" != "uosmo" ]; then
            echo $token1
        else
            echo "Both tokens are uosmo"
        fi
    elif [ "$pool_type" = "$CW_POOL_TYPE" ]; then
        contract_address=$(echo "$response" | jq -r '.pool.contract_address')
        cw_response=$(osmosisd query wasm contract-state smart $contract_address "$QUERY_MSG" -o json 2>&1)
        denom=""
        if echo "$cw_response" | grep -q "Error parsing into"; then
            cw_response=$(osmosisd query wasm contract-state smart $contract_address "$QUERY_MSG_2" -o json 2>&1)
            echo "contract_address: $contract_address" >&2
            echo "cw_response: $cw_response" >&2
            denom=$(echo "$cw_response" | jq -r '.data.total_pool_liquidity[] | select(.denom != "uosmo") | .denom' | head -n 1)
        else
            denom=$(echo "$cw_response" | jq -r '.data.assets[] | select(.info.native_token.denom != "uosmo") | .info.native_token.denom' | head -n 1)
        fi
        echo "CW Denom: $denom" >&2
        echo $denom
    else
        echo "Unknown pool type: $pool_type"
    fi
}

swap_on_pool() {
    pool_id=$1
    increment=$2
    non_osmo_denom=$(get_non_osmo_pool_asset $pool_id)
    sequence_to_use=$((SEQUENCE + increment))
    echo "SEQUENCE being used: $sequence_to_use" >&2
    output=$(osmosisd tx poolmanager swap-exact-amount-in 100000uosmo 1 --swap-route-pool-ids $pool_id --swap-route-denoms $non_osmo_denom $TX_FLAGS -s $sequence_to_use)
    echo "output: $output" >&2
    txhash=$(echo $output | jq -r '.txhash')
    echo $txhash
}

# Example usage
for ((i=0; i<${#POOL_IDS[@]}; i++)); do
    wait_for_next_block
    txhashes=()
    for ((j=0; j<=i; j++)); do
        txhash=$(swap_on_pool "${POOL_IDS[j]}" $((j)))
        txhashes+=($txhash)
    done
    echo "Round $((i+1)) txhashes: ${txhashes[@]}"
    echo "Count: ${#txhashes[@]}"
done
