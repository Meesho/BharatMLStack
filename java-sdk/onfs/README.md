# Feature Store Java SDK + Flink 1.x Connector

This repository provides the Feature Store Java SDK along with a Flink 1.x connector for writing `FeatureEvent` objects into Kafka using Protobuf serialization.

It contains:

- A framework-agnostic core library
- A Flink 1.20.x connector (using `KafkaSink`)

## Modules Overview

### 1. `feature-store-core` (Framework-agnostic)

Contains all fundamental models:

- **`FeatureEvent`** — POJO describing an entity + FG + features
- **`FeatureConverter`** — Converts `FeatureEvent` → Protobuf `persist.Query`

**Key Features:**
- Pure Java
- Strict Protobuf-only serialization
- No Kafka / Flink dependency
- Can be used in any JVM application

### 2. `feature-store-flink-connector-sdk-flink1x`

Connector module for Flink 1.20.x.

**Includes:**
- **`FeatureEventSerialization`** — converts `FeatureEvent` → Protobuf bytes
- **`FeatureEventKeySerialization`** — extracts Kafka key from `EntityLabel`
- **`KafkaSinkFactory`** — builds a ready-to-use `KafkaSink<FeatureEvent>`
- **`FeatureEventValidation`** — validates events against Horizon source mapping
- **`HorizonClient`** (in `horizon` package) — client for fetching feature mappings from Horizon API with configurable timeouts
- **`SourceMappingHolder`** (in `horizon` package) — thread-safe instance-based holder for source mapping response (supports multiple independent jobs)
- **`EtcdWatcher`** (in `etcd` package) — monitors etcd for schema changes and automatically refreshes data in a SourceMappingHolder instance

This simplifies class loading and ensures that Flink controls all Kafka behavior.

## Building the SDK

### Build all modules

```bash
cd java-sdk/onfs
mvn clean package
```

**Artifacts generated:**
- `feature-store-core/target/feature-store-core-1.0.0.jar`
- `feature-store-flink-connector-sdk-flink1x/target/feature-store-flink-connector-sdk-flink1x-1.0.0.jar`

### Override Java version

```bash
mvn -Djava.version=17 clean package
```

All modules honor the same property.

## Using the SDK in a Flink Project

### Add core dependency:

```xml
<dependency>
  <groupId>com.bharatml</groupId>
  <artifactId>feature-store-core</artifactId>
  <version>1.0.0</version>
</dependency>
```

### Add Flink connector:

```xml
<dependency>
  <groupId>com.bharatml</groupId>
  <artifactId>feature-store-flink-connector-sdk-flink1x</artifactId>
  <version>1.0.0</version>
</dependency>
```

**Flink dependencies are provided** — Make sure to keep:

```xml
<scope>provided</scope>
```

## Creating a Kafka Sink

```java
KafkaSink<FeatureEvent> sink = KafkaSinkFactory.create(
    cfg.getBootstrapServers(),
    cfg.getTopic(),
    cfg.getProducerProperties(),
    cfg.isTransactional(),
    "flink-feature-store-"
);
```

**Where:**
- `producerProperties` can include compressed, batching, retries, etc.
- If `transactional=true`, the sink uses `EXACTLY_ONCE`
- Otherwise, defaults to `AT_LEAST_ONCE`

## Using It in a Flink Job

### With Feature Mapping Validation (Recommended)

Use `FeatureEventValidation` as a filter to automatically validate all events against Horizon before sinking. Invalid events are filtered out from the stream.

**Important:** `FeatureEventValidation` uses a registry pattern to avoid Flink serialization issues. You must register the `SourceMappingHolder` with a unique key before creating the filter.

```java
import com.bharatml.featurestore.connector.FeatureEventValidation;
import com.bharatml.featurestore.connector.horizon.HorizonClient;
import com.bharatml.featurestore.connector.horizon.SourceMappingHolder;
import com.bharatml.featurestore.connector.etcd.EtcdWatcher;

DataStream<FeatureEvent> stream = ...;

// Create HorizonClient with configurable timeouts
int connectTimeoutMs = 3000;  // Connection timeout in milliseconds
int responseTimeoutMs = 1500; // Response timeout in milliseconds
HorizonClient horizonClient = new HorizonClient(horizonBaseUrl, connectTimeoutMs, responseTimeoutMs);

// Create a SourceMappingHolder instance for this job
SourceMappingHolder sourceMappingHolder = new SourceMappingHolder();

// Initialize metadata from Horizon
sourceMappingHolder.update(horizonClient.getHorizonResponse(jobId, jobToken));

// Register the holder with a unique key (e.g., jobId)
// This must be done before creating the Flink filter
String holderKey = jobId; // Use a unique identifier for this job
FeatureEventValidation.registerHolder(holderKey, sourceMappingHolder);

// Optionally: Start etcd watcher to automatically refresh metadata on schema changes
EtcdWatcher etcdWatcher = EtcdWatcher.builder()
    .etcdEndpoint(etcdEndpoint)
    .etcdUsername(etcdUsername)
    .etcdPassword(etcdPassword)
    .watchPath(watchPath)  // etcd path prefix to watch (e.g., "/config/orion-v2")
    .horizonClient(horizonClient)
    .jobId(jobId)
    .jobToken(jobToken)
    .sourceMappingHolder(sourceMappingHolder)
    .build();
etcdWatcher.start();

// Apply validation filter - pass only the key, not the holder instance
// The filter will retrieve the holder from the registry in its open() method
DataStream<FeatureEvent> validatedStream = stream
    .filter(new FeatureEventValidation(holderKey))
    .name("FeatureEventValidationFilter");

// Create Kafka sink
KafkaSink<FeatureEvent> sink = KafkaSinkFactory.create(
    cfg.getBootstrapServers(),
    cfg.getTopic(),
    cfg.getProducerProperties(),
    cfg.isTransactional(),
    "flink-feature-store-"
);

// Sink validated stream
validatedStream.sinkTo(sink);
```

**For `EXACTLY_ONCE` semantics:**

```java
env.enableCheckpointing(60000);
```

**Validation Features:**
- ✅ Validates entity label, keys schema, feature group, and feature labels against Horizon mapping
- ✅ Invalid events are filtered out (removed from stream)
- ✅ Supports multiple independent jobs (instance-based `SourceMappingHolder` with registry pattern)
- ✅ Dynamic schema updates via `EtcdWatcher` without job restart
- ✅ Metrics: Access valid and invalid events counts via `getValidEventsCount()` and `getInvalidEventsCount()` instance methods respectively.

## Dynamic Schema Updates with EtcdWatcher

`EtcdWatcher` monitors etcd for schema changes and automatically refreshes the source mapping. The watcher watches `{watchPath}/entities` with prefix mode enabled.

**Note:** Pass the base path (e.g., `"/config/orion-v2"`) as `watchPath` - the code automatically appends `/entities`.

```java
EtcdWatcher etcdWatcher = EtcdWatcher.builder()
    .etcdEndpoint(etcdEndpoint)
    .etcdUsername(etcdUsername)
    .etcdPassword(etcdPassword)
    .watchPath("/config/orion-v2")  // Base path - /entities is appended automatically
    .horizonClient(horizonClient)
    .jobId(jobId)
    .jobToken(jobToken)
    .sourceMappingHolder(sourceMappingHolder)
    .build();
etcdWatcher.start();
```

The watcher automatically restarts on errors/completion and updates the `SourceMappingHolder` when changes are detected.



## Testing

Run unit tests:

```bash
cd java-sdk/onfs/feature-store-core
mvn test
```

The core module includes `FeatureConverterTest` which verifies Protobuf serialization.

## Compatibility

- **Java 11+** (default, configurable via `java.version` property)
- **Flink 1.20.1** (connector module)
- **Kafka 3.5.0** (kafka-clients)
- **Protobuf 3.24.1**

## Limitations

- **Flink 1.x only**: The connector targets Flink 1.20.1. For Flink 2.x, a separate connector module would be needed.
- **Kafka only**: Currently only supports Kafka as the sink backend.
- **Protobuf-only**: No JSON serialization support.
