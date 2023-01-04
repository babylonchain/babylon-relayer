#!/usr/bin/env sh

INTERVAL=${1:-5m}

echo "Start relaying headers of all chains in the config to Babylon with interval $INTERVAL"

nohup babylon-relayer keep-update-clients --interval $INTERVAL > relayer-all.log &