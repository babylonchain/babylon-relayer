# babylon-relayer

Babylon relayer is an extended version of the IBC relayer specialised for timestamping Cosmos SDK chains.

## Requirements

- Go 1.19

## Development requirements

- Go 1.19

## Building

To build the chain, simply:
```console
make build
```

This will lead to the creation of a `babylon-relayer` executable under the `build/` directory.

## Installing

To build the chain and install a babylon executable:
```console
make install
```

## Testing

```console
make test
```

## Configuration

The configuration of Babylon relayer is exactly the same as the official IBC relayer.
Please read [the IBC relayer's documentation](https://github.com/cosmos/relayer/tree/main/docs).
This repo also provides some example configurations under the `example/` directory.

## Usage

To add chains to `config.yaml`:
```console
babylon-relayer chains add-dir examples/chains
```

To add paths to `config.yaml`:
```console
babylon-relayer paths add-dir examples/paths
```

To restore secret keys from mnenomics:
```console
babylon-relayer keys restore $CHAIN $KEY_NAME $MNEMONICS
```

To create an IBC light client for a chain in Babylon:
```console
babylon-relayer tx client babylon $CHAIN $CHAIN
```

To start relaying headers of a chain to Babylon:
```console
babylon-relayer keep-update-client babylon $CHAIN $CHAIN --interval $INTERVAL
```
