# Feature Store Flink Connector SDK (Flink 1.x)

Flink connector module for Flink 1.20.1 that provides two sink implementations for publishing FeatureEvent messages to Kafka with Protobuf serialization.

## Features

- **FeatureStoreKafkaSink** - Simple sink with at-least-once semantics (RichSinkFunction)
- **FeatureStoreTransactionalKafkaSink** - Transactional sink with exactly-once semantics (TwoPhaseCommitSinkFunction)
- **FeatureStoreClientConfig** - Builder pattern for configuration with optional KafkaProducerFactory injection
- **FeatureSinkFactory** - Factory methods to create sinks

## Building

This module is self-contained with all properties defined in its `pom.xml`:

```bash
cd feature-store-flink-connector-sdk-flink1x
mvn clean package
```

### Override Java Version

Default Java version is 11. To build with a different version:

```bash
mvn -Djava.version=17 clean package
```

**Note:** Flink dependencies are marked as `<scope>provided</scope>`, so the connector jar does NOT bundle Flink runtime jars. The Flink runtime must supply these dependencies.

## Dependencies

- **Java 11+** (default, configurable via `java.version` property)
- **Flink 1.20.1** (provided scope - supplied by Flink runtime)
- `feature-store-core` (normal scope - included in connector jar)
- `feature-store-kafka-client` (normal scope - included in connector jar)
- `org.apache.kafka:kafka-clients` (3.5.0, normal scope)

## Usage

### Simple Sink (At-Least-Once)

```java
import com.bharatml.featurestore.connector.*;
import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.streaming.api.datastream.DataStream;

FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()
    .bootstrapServers("localhost:9092")
    .topic("my-topic")
    .acks("all")
    .retries(3)
    .build();

DataStream<FeatureEvent> events = ...;
events.addSink(new FeatureStoreKafkaSink(config));
// Or use factory:
// events.addSink(FeatureSinkFactory.createSink(config));
```

### Transactional Sink (Exactly-Once)

```java
import com.bharatml.featurestore.connector.*;
import com.bharatml.featurestore.core.FeatureEvent;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.datastream.DataStream;

StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
env.enableCheckpointing(60000); // Required for exactly-once

FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()
    .bootstrapServers("localhost:9092")
    .topic("my-topic")
    .transactional(true)
    .build();

DataStream<FeatureEvent> events = ...;
events.addSink(new FeatureStoreTransactionalKafkaSink(config));
// Or use factory:
// events.addSink((TwoPhaseCommitSinkFunction) FeatureSinkFactory.createTransactionalKafkaSink(config));
```

### Custom KafkaProducerFactory

```java
import com.bharatml.featurestore.kafka.KafkaProducerFactory;

// Create custom factory
KafkaProducerFactory customFactory = new CustomKafkaProducerFactory();

FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()
    .bootstrapServers("localhost:9092")
    .topic("my-topic")
    .producerFactory(customFactory)  // Inject custom factory
    .build();
```

### Configuration Builder

```java
FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()
    .bootstrapServers("localhost:9092")
    .topic("my-topic")
    .acks("all")                    // Default: "all"
    .retries(3)                     // Default: 3
    .transactional(false)          // Default: false
    .additionalProperty("compression.type", "snappy")
    .additionalProperty("batch.size", "32768")
    .additionalProperty("linger.ms", "10")  // TODO: Tune for production throughput
    .build();
```

## Sink Comparison

| Feature | FeatureStoreKafkaSink | FeatureStoreTransactionalKafkaSink |
|---------|----------------------|-------------------------------|
| **Semantics** | At-least-once | Exactly-once |
| **Implementation** | `RichSinkFunction` | `TwoPhaseCommitSinkFunction` |
| **Checkpointing** | Not required | Required |
| **Use Case** | Simple demos, debugging | Production with strict guarantees |
| **Performance** | Lower overhead | Slightly higher overhead |
| **Flink Version** | 1.20.1 | 1.20.1 |
| **Kafka Transactions** | Not used | Required |

## Transactional ID Generation

**Important:** The current implementation uses UUID for transactional IDs, which may cause issues on task restart. 

For production, modify `FeatureStoreTransactionalKafkaSink.beginTransaction()` to generate deterministic transactional IDs using:
- Job ID (from RuntimeContext)
- Subtask index (from RuntimeContext)
- Format: `"flink-feature-store-{jobId}-{subtaskIndex}"`

This ensures idempotency across job restarts and prevents transaction coordinator conflicts.

## Deployment

### Flink Job JAR

Include the connector jar in your Flink job's classpath:

```bash
# Option 1: Copy to Flink lib directory (for all jobs)
cp feature-store-flink-connector-sdk-flink1x/target/feature-store-flink-connector-sdk-flink1x-1.0.0.jar \
   $FLINK_HOME/lib/

# Option 2: Include in job fat jar (shade plugin)
# Add connector as dependency in your job's pom.xml
```

### Flink Cluster

Ensure Flink 1.20.1 is installed and running. The connector will use Flink's provided dependencies.

## Production Tuning

TODO comments in code indicate areas for production tuning:
- **Transactional ID**: Use deterministic IDs (jobId + subtaskIndex)
- **Producer retries/backoff**: Configure via `additionalProperty("retries", "...")`
- **Idempotence**: Enabled by default in DefaultKafkaProducerFactory
- **Acks**: Default "all" for reliability
- **linger.ms**: Tune for throughput vs latency tradeoff
- **batch.size**: Tune for throughput

## Limitations

- **Flink 1.x only**: This connector targets Flink 1.20.1. For Flink 2.x, a separate connector module would be needed.
- **Kafka only**: Currently only supports Kafka as the sink backend.
- **Protobuf-only**: No JSON serialization support.

