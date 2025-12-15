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
- **`HorizonClient`** (in `horizon` package) — client for fetching feature mappings from Horizon API
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

// Create HorizonClient
HorizonClient horizonClient = new HorizonClient(horizonBaseUrl);

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
    .watchKey(watchKey)
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

### Without Validation (Not Recommended)

```java
DataStream<FeatureEvent> stream = ...;
KafkaSink<FeatureEvent> sink = KafkaSinkFactory.create(
    cfg.getBootstrapServers(),
    cfg.getTopic(),
    cfg.getProducerProperties(),
    cfg.isTransactional(),
    "flink-feature-store-"
);
stream.sinkTo(sink);
```

**For `EXACTLY_ONCE`:**

```java
env.enableCheckpointing(60000);
```

**Validation Features:**
- ✅ Validates entity label exists in Horizon mapping
- ✅ Validates keys schema matches Horizon expectations
- ✅ Validates feature group label is registered
- ✅ Validates feature labels are registered for the entity/feature group combination
- ✅ **Invalid events are filtered out** (removed from stream, not causing failures)
- Validation mapping is read from the `SourceMappingHolder` instance, which can be updated dynamically via `EtcdWatcher`
- ✅ **Supports multiple independent jobs**: Each job can have its own `SourceMappingHolder` instance with different `jobId` and `jobToken`
- ✅ **Flink-compatible**: Uses registry pattern to avoid serialization issues with Flink operators

## Dynamic Source Mapping Updates with EtcdWatcher

The `EtcdWatcher` monitors an etcd key prefix for changes and automatically refreshes the source mapping from Horizon when changes are detected. This allows the validation to use the latest schema without restarting the Flink job.

```java
// Create HorizonClient
HorizonClient horizonClient = new HorizonClient(horizonBaseUrl);

// Create a SourceMappingHolder instance
SourceMappingHolder sourceMappingHolder = new SourceMappingHolder();

// Initialize with current schema
sourceMappingHolder.update(horizonClient.getHorizonResponse(jobId, jobToken));

// Start etcd watcher to monitor schema changes using builder pattern
EtcdWatcher etcdWatcher = EtcdWatcher.builder()
    .etcdEndpoint(etcdEndpoint)      // etcd server endpoint
    .etcdUsername(etcdUsername)      // etcd username
    .etcdPassword(etcdPassword)      // etcd password
    .watchKey(watchKey)              // etcd key prefix to watch (e.g., "/config/orion-v2/entities")
    .horizonClient(horizonClient)    // HorizonClient instance
    .jobId(jobId)                    // Job ID for Horizon API
    .jobToken(jobToken)              // Job token for Horizon API
    .sourceMappingHolder(sourceMappingHolder) // SourceMappingHolder instance to update
    .build();
etcdWatcher.start();

// The watcher will automatically update the SourceMappingHolder instance when etcd changes are detected
// FeatureEventValidation will use the updated schema for subsequent events
```

## Multiple Jobs Support

Since `SourceMappingHolder` is instance-based and `FeatureEventValidation` uses a registry pattern, you can run multiple Flink jobs with different `jobId` and `jobToken` values, each maintaining its own independent source mapping:

```java
// Job 1 with its own source mapping
HorizonClient client1 = new HorizonClient(horizonBaseUrl);
SourceMappingHolder holder1 = new SourceMappingHolder();
holder1.update(client1.getHorizonResponse(jobId1, jobToken1));
FeatureEventValidation.registerHolder("job1", holder1);
EtcdWatcher watcher1 = EtcdWatcher.builder()
    .etcdEndpoint(etcdEndpoint)
    .etcdUsername(etcdUsername)
    .etcdPassword(etcdPassword)
    .watchKey(watchKey)
    .horizonClient(client1)
    .jobId(jobId1)
    .jobToken(jobToken1)
    .sourceMappingHolder(holder1)
    .build();
FeatureEventValidation validation1 = new FeatureEventValidation("job1");

// Job 2 with its own independent source mapping
HorizonClient client2 = new HorizonClient(horizonBaseUrl);
SourceMappingHolder holder2 = new SourceMappingHolder();
holder2.update(client2.getHorizonResponse(jobId2, jobToken2));
FeatureEventValidation.registerHolder("job2", holder2);
EtcdWatcher watcher2 = EtcdWatcher.builder()
    .etcdEndpoint(etcdEndpoint)
    .etcdUsername(etcdUsername)
    .etcdPassword(etcdPassword)
    .watchKey(watchKey)
    .horizonClient(client2)
    .jobId(jobId2)
    .jobToken(jobToken2)
    .sourceMappingHolder(holder2)
    .build();
FeatureEventValidation validation2 = new FeatureEventValidation("job2");
```

## Protobuf Schema

**Located at:**
- `java-sdk/onfs/proto/feature.proto`

**Generated at:**
- `target/generated-sources/protobuf/java/persist/`

## Feature Mapping Validation

The SDK provides validation of feature mappings against Horizon before events are written to Kafka:

- ✅ Validates entity label exists in Horizon mapping
- ✅ Validates keys schema matches Horizon expectations
- ✅ Validates feature group label is registered
- ✅ Validates feature labels are registered for the entity/feature group combination
- ✅ **Invalid events are filtered out** (removed from stream)
- ✅ **Supports multiple independent jobs**: Each job maintains its own source mapping
- ✅ **Flink-compatible**: Uses registry pattern to avoid serialization issues

This prevents ingestion of invalid or unregistered features. The validation uses a `SourceMappingHolder` instance which can be updated dynamically via `EtcdWatcher` to reflect schema changes without restarting the Flink job.

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
