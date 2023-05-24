# babylon-relayer

Babylon relayer is an extended version of the IBC relayer specialised for timestamping Cosmos SDK chains.

## Development requirements

- Go 1.20

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

Note that some chains (e.g., Injective and EVMOS) impose extra codec formats for its RPC calls.
To support such chains, one needs to add an `"extra-codecs"` entry to its config json file.
An example can be found in `examples/chains/injective.json`.

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

To start relaying headers of all chains in the config to Babylon:
```console
babylon-relayer --home /home/ubuntu/data/relayer keep-update-clients --interval $INTERVAL
```
