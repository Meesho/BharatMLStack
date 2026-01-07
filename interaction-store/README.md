# Interaction Store

A high-performance ScyllaDB-based system for storing and querying user interaction data (clicks, orders, wishlist, etc.) with sub-10ms latency guarantees. Part of the [BharatML Stack](https://github.com/Meesho/BharatMLStack).

## Features

- **Last Y Days**: Retrieve all interactions from last Y days (up to 30 days / 2,000 interactions)
- **Weekly Time-Bucketed Storage**: Data partitioned into weekly columns with ZSTD compression
- **Vertical Sharding**: Tables split into 8-week chunks for query isolation and efficient compaction

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                      Interaction Store                           │
├──────────────────────────────────────────────────────────────────┤
│  cmd/serving         │  cmd/consumers                            │
│  (gRPC + HTTP API)   │  (Kafka Consumer)                         │
├──────────────────────────────────────────────────────────────────┤
│  Handlers: Click | Order                                         │
│  ├── Persist: Write interactions to storage                      │
│  └── Retrieve: Query interactions by time range                  │
├──────────────────────────────────────────────────────────────────┤
│  Data Layer                                                      │
│  ├── PSDB (Permanent Storage Data Block) with ZSTD compression   │
│  └── ScyllaDB (RF=2, STCS Compaction)                            │
│      ┌──────────────┬───────────────┬───────────────┬──────────┐ │
│      │ week_1_to_8  │ week_9_to_16  │ week_17_to_24 │ metadata │ │
│      └──────────────┴───────────────┴───────────────┴──────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### gRPC Service

```protobuf
service TimeSeriesService {
    rpc PersistClickData(PersistClickDataRequest) returns (PersistDataResponse);
    rpc PersistOrderData(PersistOrderDataRequest) returns (PersistDataResponse);
    rpc RetrieveClickInteractions(RetrieveDataRequest) returns (RetrieveClickDataResponse);
    rpc RetrieveOrderInteractions(RetrieveDataRequest) returns (RetrieveOrderDataResponse);
    rpc RetrieveInteractions(RetrieveInteractionsRequest) returns (RetrieveInteractionsResponse);
}
```