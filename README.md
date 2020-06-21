# Blockbook/RUBTC

This is a fork of back-end service for Trezor wallet. Main features are:

- index of addresses and address balances of the connected block chain
- fast searches in the indexes
- simple blockchain explorer
- websocket, API and legacy Bitcore Insight compatible socket.io interfaces

## Build and installation instructions

Officially supported platform is **Debian Linux** and **AMD64** architecture.

Developer build guide is [here](/docs/build.md).

## Implemented coins

This fork is focused on Russian Bitcoin support. No other coins are implemented.

## Common issues when running Blockbook or implementing additional coins

#### Out of memory when doing initial synchronization

How to reduce memory footprint of the initial sync: 

- disable rocksdb cache by parameter `-dbcache=0`, the default size is 500MB
- run blockbook with parameter `-workers=1`. This disables bulk import mode, which caches a lot of data in memory (not in rocksdb cache). It will run about twice as slowly but especially for smaller blockchains it is no problem at all.

#### Error `internalState: database is in inconsistent state and cannot be used`

Blockbook was killed during the initial import, most commonly by OOM killer. By default, Blockbook performs the initial import in bulk import mode, which for performance reasons does not store all the data immediately to the database. If Blockbook is killed during this phase, the database is left in an inconsistent state. 

See above how to reduce the memory footprint, delete the database files and run the import again.

## Data storage in RocksDB

Blockbook stores data the key-value store RocksDB. Database format is described [here](/docs/rocksdb.md).

## API

Blockbook API is described [here](/docs/api.md).
