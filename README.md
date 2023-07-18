# AVAX Indexer

## Overview

This indexer is a tool to index the AVAX blockchain and store the data in a MongoDB database. It stores the last 10000 blocks utilizing MongoDB [Capped Collections](https://www.mongodb.com/docs/manual/core/capped-collections/).

## Environment Variables

| Name              | Description                                        | Default                                           |
|-------------------|----------------------------------------------------|---------------------------------------------------|
| `MONGODB_URI`     | MongoDB connection string                          | None                                              |
| `AVAX_RPC`        | RPC endpoint for the Avalanche network             | `https://api.avax.network/ext/bc/C/rpc` (mainnet) |
| `AVAX_RPC_INFURA` | RPC endpoint for the Avalanche network from Infura | None                                              |
| `AVAX_WS`         | WS endpoint for the Avalanche network              | `wss://api.avax.network/ext/bc/C/ws` (mainnet)    |


