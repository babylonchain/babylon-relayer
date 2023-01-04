#!/usr/bin/env sh

CHAIN=${1:-osmosis}

echo "Creating $CHAIN light client on Babylon..."

babylon-relayer tx client babylon $CHAIN $CHAIN