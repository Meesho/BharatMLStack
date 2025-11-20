# Feature Store Kafka Client

Kafka producer factory and utilities for Feature Store. Provides abstraction for creating KafkaProducer instances.

## Features

- **KafkaProducerFactory** - Interface for creating KafkaProducer instances
- **DefaultKafkaProducerFactory** - Default implementation with standard configuration
- **ProducerWrapper** - Helper utilities for sending records with callbacks

## Building

This module is self-contained with all properties defined in its `pom.xml`:

```bash
cd feature-store-kafka-client
mvn clean package
```

### Override Java Version

Default Java version is 11. To build with a different version:

```bash
mvn -Djava.version=17 clean package
```

## Dependencies

- **Java 11+** (default, configurable via `java.version` property)
- `org.apache.kafka:kafka-clients` (3.5.0)

## Usage

### Adding as Dependency

```xml
<dependency>
    <groupId>com.bharatml</groupId>
    <artifactId>feature-store-kafka-client</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Using Default Factory

```java
import com.bharatml.featurestore.kafka.*;

Properties props = new Properties();
props.setProperty("bootstrap.servers", "localhost:9092");

KafkaProducerFactory factory = new DefaultKafkaProducerFactory();

// Non-transactional producer
KafkaProducer<String, byte[]> producer = 
    factory.createProducer(props, false, null);

// Transactional producer
KafkaProducer<String, byte[]> transactionalProducer = 
    factory.createProducer(props, true, "my-transactional-id");
```

### Custom Factory Implementation

Implement `KafkaProducerFactory` for custom producer creation logic:

```java
public class CustomKafkaProducerFactory implements KafkaProducerFactory {
    @Override
    public KafkaProducer<String, byte[]> createProducer(
            Properties props, boolean transactional, String transactionalId) {
        // Custom implementation
        return new KafkaProducer<>(props);
    }
}
```

### Using ProducerWrapper

```java
import com.bharatml.featurestore.kafka.*;

ProducerRecord<String, byte[]> record = new ProducerRecord<>("topic", "key", data);

// Async send with callback
Future<RecordMetadata> future = ProducerWrapper.sendWithCallback(producer, record);

// Blocking send
RecordMetadata metadata = ProducerWrapper.sendBlocking(producer, record);
```

## Configuration

The `DefaultKafkaProducerFactory` sets the following defaults:
- `acks=all` - Wait for all replicas
- `retries=3` - Retry failed sends
- `enable.idempotence=true` - Prevent duplicate messages
- For transactional: `max.in.flight.requests.per.connection=5`

Additional properties can be provided via the Properties object passed to `createProducer()`.

