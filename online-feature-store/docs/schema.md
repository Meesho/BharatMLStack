# Key-Schema Isolation and Mapping Using etcd

## Overview
In the PSDB-based feature storage architecture, keys (representing entity or feature group identifiers) are decoupled from their corresponding data values. This separation is critical for enabling fast, flexible lookups, versioned schema evolution, and cache coherency. The mapping between keys and their corresponding feature group schema is maintained via etcd, a distributed key-value store optimized for consistency and availability.

## Key Concepts

### 1. **Key-Value Isolation**
Keys and data values are handled separately:
- **Keys** are used to locate data blobs stored in Redis, ScyllaDB, CSDB, or other backends.
- **Values** are binary-encoded PSDBs containing data serialized using a specific schema version.

This separation allows the system to:
- Quickly redirect lookups without re-reading bulky data
- Change schema or compressions without modifying key structure
- Minimize duplication in storage and caching layers

### 2. **etcd for Schema Mapping**
Each key prefix in etcd maintains metadata:
- **Feature Group ID → Schema Version Mapping**
- **Feature Group ID → TTL or other config metadata**

When a PSDB is read, its header contains the `FeatureSchemaVersion`. This version is used to retrieve the corresponding schema definition from etcd, which describes:
- Feature names
- Data types
- Vector shapes (if applicable)

### 3. **Schema Version Usage**
Each time a feature group is updated (e.g., feature added or deleted), a new schema version is generated and stored in etcd. This version is embedded into the PSDB header during serialization. During deserialization:
1. The PSDB header is parsed to extract the `FeatureSchemaVersion`
2. etcd is queried to fetch the schema for that version
3. The schema is used to correctly interpret the deserialized binary data

## Storage Model

### 🔹 Chunked Value Storage in ScyllaDB
- The serialized PSDB `[]byte` is chunked into 1KB segments
- These chunks are stored across multiple columns (e.g., `c0`, `c1`, `c2`, ...) in ScyllaDB or permanent storage
- Chunking keeps column sizes consistent and allows efficient partial reads/writes

This enables:
- Better memory management on read/write paths
- Reduced chances of tombstone explosion in wide-column stores like ScyllaDB

### 🔹 Wide Table Support with StoreID
To prevent ScyllaDB tables from becoming unmanageably large:
- The system introduces a `storeId` abstraction
- When the number of feature groups mapped to a table grows, a new feature group is assigned a new `storeId`
- This maps to a new underlying table or column family

This strategy helps:
- Distribute data evenly across tables
- Prevent schema bloat and performance degradation
- Simplify horizontal scaling of feature group storage

## Benefits of This Design

### 🔄 **Schema Evolution Support**
- New features can be added or deprecated without rewriting existing data
- Allows backward and forward compatibility through schema versioning

### ⚡ **Low-Latency Reads**
- Keys are compact and isolated from bulky values, enabling fast routing
- Only values are fetched and deserialized when necessary

### 🧠 **Intelligent Caching**
- Schema definitions are small and cached in memory
- Reduces the need to re-parse schema with each request

### 🔍 **Version-Aware Decoding**
- Prevents corruption or misinterpretation due to stale schema assumptions
- Enables safe replay of historical data with the correct schema

### 🛠️ **Operational Simplicity**
- Centralized schema mapping in etcd makes management auditable
- Easy to observe changes, roll back versions, or audit usage

### 🧱 **Scalable Storage Model**
- Chunked columnar storage avoids large row hotspots and enables elastic scaling
- Wide table sharding via `storeId` keeps Scylla tables lean and performant

## Summary
The separation of keys and values, combined with schema-version mapping through etcd, forms a robust foundation for scalable and evolvable feature storage. It ensures data consistency, minimizes overhead, and unlocks the ability to handle complex schema changes over time—all without impacting data correctness or system performance.

The chunked storage strategy and use of `storeId` for table isolation further reinforce the system’s scalability, enabling it to support thousands of feature groups with predictable performance across storage layers.

