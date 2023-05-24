#!/usr/bin/env sh
set -euo pipefail
#set -x

# 0. Define configuration
BABYLON_KEY="babylon-key"
BABYLON_CHAIN_ID="chain-test"
GAIA_CHAIN_ID="gaia-test"

mkdir -p $RELAYER_CONF_DIR
babylon-relayer --home $RELAYER_CONF_DIR config init
RELAYER_CONF=$RELAYER_CONF_DIR/config/config.yaml

cat <<EOT > $RELAYER_CONF
global:
    api-listen-addr: :5183
    timeout: 20s
    memo: ""
    light-cache-size: 10
chains:
    babylon:
        type: cosmos
        value:
            key: $BABYLON_KEY
            chain-id: $BABYLON_CHAIN_ID
            rpc-addr: $BABYLON_NODE_RPC
            account-prefix: bbn
            keyring-backend: test
            gas-adjustment: 1.5
            gas-prices: 0.002ubbn
            min-gas-amount: 1
            debug: true
            timeout: 10s
            output-format: json
            sign-mode: direct
            extra-codecs: []
    gaia:
        type: cosmos
        value:
            chain-id: $GAIA_CHAIN_ID
            rpc-addr: http://localhost:26657
            keyring-backend: test
            timeout: 10s            
EOT

# 1. Create a gaiad testnet

# Create testnet dirs for one validator
echo "Creating testnet dirs..."
gaiad testnet \
    --v                     1 \
    --output-dir            $GAIA_CONF \
    --starting-ip-address   192.168.10.2 \
    --keyring-backend       test \
    --minimum-gas-prices    "0.00002stake" \
    --chain-id              $GAIA_CHAIN_ID

echo "$(sed 's/cors_allowed_origins = \[\]/cors_allowed_origins = \[\"*\"\]/g' $GAIA_CONF/node0/gaiad/config/config.toml)" > $GAIA_CONF/node0/gaiad/config/config.toml
# Start the gaiad service
echo "Starting the gaiad service..."
GAIA_LOG=$GAIA_CONF/node0/gaiad/gaiad.log
gaiad --home $GAIA_CONF/node0/gaiad start \
      --pruning=nothing --grpc-web.enable=false \
      --rpc.unsafe true \
      --grpc.address="0.0.0.0:9091" > $GAIA_LOG 2>&1 &

echo "gaiad started. Logs outputted at $GAIA_LOG"
sleep 10
echo "Status of Gaia node"
gaiad status

sleep 15

# 2. Create the relayer
echo "Inserting Babylon key"
BABYLON_MEMO=$(cat $BABYLON_HOME/key_seed.json | jq .secret | tr -d '"')
babylon-relayer --home $RELAYER_CONF_DIR keys restore babylon $BABYLON_KEY "$BABYLON_MEMO"

echo "Start the Babylon relayer"
babylon-relayer --home $RELAYER_CONF_DIR keep-update-clients --interval $UPDATE_CLIENTS_INTERVAL
