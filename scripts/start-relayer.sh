#!/usr/bin/env sh

CHAIN=${1:-osmosis}
INTERVAL=${2:-5m}

echo "Start relaying $CHAIN headers to Babylon with interval $INTERVAL"

nohup babylon-relayer keep-update-client babylon $CHAIN $CHAIN --interval $INTERVAL > relayer-$CHAIN.log &