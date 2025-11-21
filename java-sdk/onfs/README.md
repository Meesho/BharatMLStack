# Feature Store Java SDK

Self-contained Java SDK for Feature Store with three independent Maven modules.

## Structure

```
java-sdk/
├── proto/
│   └── feature.proto                          # Protobuf schema
├── feature-store-core/                        # Core library (NO Kafka or Flink dependencies)
├── feature-store-kafka-client/                # Kafka producer factory
└── feature-store-flink-connector-sdk-flink1x/ # Flink 1.20.1 connector
```

## Modules

### feature-store-core

**Framework-agnostic core library** that can be published as a standalone artifact.

- **FeatureEvent** - POJO for feature events
- **FeatureConverter** - Converts FeatureEvent to Protobuf Query (strictly Protobuf-only)

**Dependencies:**
- `com.google.protobuf:protobuf-java` (3.24.1)

### feature-store-kafka-client

**Kafka producer factory and utilities.**

- **KafkaProducerFactory** - Interface for creating KafkaProducer instances
- **DefaultKafkaProducerFactory** - Default implementation
- **ProducerWrapper** - Helper utilities for sending records

**Dependencies:**
- `org.apache.kafka:kafka-clients` (3.5.0)

### feature-store-flink-connector-sdk-flink1x

**Flink 1.20.1 connector** with two sink implementations.

- **FeatureStoreKafkaSink** - Simple sink (at-least-once) using `RichSinkFunction`
- **FeatureStoreTransactionalSink** - Transactional sink (exactly-once) using `TwoPhaseCommitSinkFunction`
- **FeatureStoreClientConfig** - Builder pattern for configuration
- **FeatureSinkFactory** - Factory methods to create sinks

**Dependencies:**
- `feature-store-core` (normal scope)
- `feature-store-kafka-client` (normal scope)
- `org.apache.flink:flink-streaming-java` (1.20.1, **provided scope**)
- `org.apache.kafka:kafka-clients` (3.5.0, normal scope)

**Important:** Flink dependencies are marked as `provided` scope, so the connector jar does NOT bundle Flink runtime jars. The Flink runtime must supply these dependencies.

## Building

### Build All Modules

Use the provided build script to build all modules (core, kafka-client, and Flink connector):

```bash
cd java-sdk/onfs
./build-local-flink-connector.sh
```

The script uses the parent POM to build all modules in the correct order. This will produce:
- `feature-store-core/target/feature-store-core-1.0.0.jar`
- `feature-store-kafka-client/target/feature-store-kafka-client-1.0.0.jar`
- **`feature-store-flink-connector-sdk-flink1x/target/feature-store-flink-connector-sdk-flink1x-1.0.0.jar`** (Flink connector JAR)

The Flink connector JAR is the main artifact needed for Flink jobs. It includes the core and kafka-client modules as dependencies.

### Override Java Version

Default Java version is 11. To build with a different version:

```bash
JAVA_VERSION=17 ./build-local-flink-connector.sh
```

Or directly using Maven:

```bash
cd java-sdk/onfs
mvn -Djava.version=17 clean package
```

Each module's `pom.xml` uses the `java.version` property for source/target/release settings.

### Build Individual Modules

Each module can also be built independently:

```bash
cd java-sdk/onfs/feature-store-core
mvn -Djava.version=11 clean package

cd ../feature-store-kafka-client
mvn -Djava.version=11 clean package

cd ../feature-store-flink-connector-sdk-flink1x
mvn -Djava.version=11 clean package
```

## Usage

### Core Module (Standalone)

```java
import com.bharatml.featurestore.core.*;

FeatureEvent event = new FeatureEvent();
// ... set event fields ...

persist.QueryOuterClass.Query query = FeatureConverter.toProto(event);
byte[] protobufBytes = query.toByteArray();
```

### Flink Connector (Simple Sink)

```java
import com.bharatml.featurestore.connector.*;
import com.bharatml.featurestore.core.FeatureEvent;

FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()
    .bootstrapServers("localhost:9092")
    .topic("my-topic")
    .build();

DataStream<FeatureEvent> events = ...;
events.addSink(new FeatureStoreKafkaSink(config));
```

### Flink Connector (Transactional Sink)

```java
import com.bharatml.featurestore.connector.*;
import com.bharatml.featurestore.core.FeatureEvent;

StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
env.enableCheckpointing(60000); // Required for exactly-once

FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()
    .bootstrapServers("localhost:9092")
    .topic("my-topic")
    .transactional(true)
    .build();

DataStream<FeatureEvent> events = ...;
events.addSink(new FeatureStoreTransactionalSink(config));
```

## Compatibility

- **Java 11+** (default, configurable via `java.version` property)
- **Flink 1.20.1** (connector module)
- **Kafka 3.5.0** (kafka-clients)
- **Protobuf 3.24.1**

## Deployment

### Publishing Core Module

The `feature-store-core` module can be published to Maven repositories as a standalone artifact since it has no external framework dependencies.

### Using Connector in Flink Jobs

**Option 1: Include in Flink lib directory (for all jobs)**
```bash
cp feature-store-flink-connector-sdk-flink1x/target/feature-store-flink-connector-sdk-flink1x-1.0.0.jar \
   $FLINK_HOME/lib/
```

**Option 2: Include in job fat jar**
Add connector as dependency in your job's `pom.xml` and use shade plugin to create fat jar.

### Flink Cluster Requirements

- Flink 1.20.1 installed and running
- Kafka broker accessible
- Checkpointing enabled (for transactional sink)

## Choosing the Right Sink

| Use Case | Sink | Semantics |
|----------|------|-----------|
| Development, debugging | `FeatureStoreKafkaSink` | At-least-once |
| Production with strict guarantees | `FeatureStoreTransactionalSink` | Exactly-once |

## Transactional ID Generation

**Important:** The current `FeatureStoreTransactionalSink` implementation uses UUID for transactional IDs, which may cause issues on task restart.

For production, modify `FeatureStoreTransactionalSink.beginTransaction()` to generate deterministic transactional IDs using task index, job ID, and subtask ID.

## Testing

Run unit tests:

```bash
cd java-sdk/feature-store-core
mvn test
```

The core module includes `FeatureConverterTest` which verifies Protobuf serialization.

## Protobuf Schema

The SDK uses `proto/feature.proto` as the canonical schema. The protobuf plugin generates Java classes during compilation.

Generated classes are in: `target/generated-sources/protobuf/java/persist/`

## Limitations

- **Flink 1.x only**: The connector targets Flink 1.20.1. For Flink 2.x, a separate connector module would be needed.
- **Kafka only**: Currently only supports Kafka as the sink backend.
- **Protobuf-only**: No JSON serialization support.

