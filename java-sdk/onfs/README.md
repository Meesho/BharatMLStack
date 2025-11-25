# Feature Store Java SDK + Flink 1.x Connector

This repository provides the Feature Store Java SDK along with a Flink 1.x connector for writing `FeatureEvent` objects into Kafka using Protobuf serialization.

It contains:

- A framework-agnostic core library
- A Flink 1.20.x connector (using `KafkaSink`)

## 🎯 Modules Overview

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

This simplifies class loading and ensures that Flink controls all Kafka behavior.

## 🛠 Building the SDK

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

## 📦 Using the SDK in a Flink Project

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

## 🧩 Creating a Kafka Sink

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

## 🚀 Using It in a Flink Job

```java
DataStream<FeatureEvent> stream = ...;
KafkaSink<FeatureEvent> sink = KafkaSinkFactory.create(...);
stream.sinkTo(sink);
```

**For `EXACTLY_ONCE`:**

```java
env.enableCheckpointing(60000);
```

## 🧪 Protobuf Schema

**Located at:**
- `java-sdk/onfs/proto/feature.proto`

**Generated at:**
- `target/generated-sources/protobuf/java/persist/`

## 🛡 TODO (Future Work)

### Feature Mapping Validation

The SDK will soon introduce strict validation of the ingested features:

- Validate entity label
- Validate feature group label
- Validate list of feature names
- Validate feature data types (fp64, bool, string, vector, etc.)
- Validate schema compliance

This will prevent ingestion of invalid or unregistered features.


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
