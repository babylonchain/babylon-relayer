#!/usr/bin/env sh
set -euo pipefail
set -x

BINARY=${BINARY:-/app/babylon-relayer}
RELAY_INTERVAL=${RELAY_INTERVAL:-6m}
RELAY_DEBUG_ADDR=${RELAY_DEBUG_ADDR:-127.0.0.1:7597}

if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found. Please add the binary to the shared folder. Please use the BINARY environment variable if the name of the binary is not 'babylon-relayer'"
	exit 1
fi

export HOME_DIR=${HOME_DIR:-/app/.relayer/}

$BINARY --home $HOME_DIR keep-update-clients --interval $RELAY_INTERVAL --debug-addr $RELAY_DEBUG_ADDR 2>&1
