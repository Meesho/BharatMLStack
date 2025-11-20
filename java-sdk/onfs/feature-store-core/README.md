# Feature Store Core

Core library with POJOs and Protobuf converters.  - this module is completely framework-agnostic and can be published as a standalone artifact.

## Features

- **FeatureEvent** - POJO representing feature events
- **FeatureConverter** - Converts FeatureEvent to Protobuf Query messages (strictly Protobuf-only, no JSON fallback)

## Building

This module is self-contained with all properties defined in its `pom.xml`:

```bash
cd feature-store-core
mvn clean package
```

### Override Java Version

Default Java version is 11. To build with a different version:

```bash
mvn -Djava.version=17 clean package
```

## Dependencies

- **Java 11+** (default, configurable via `java.version` property)
- `com.google.protobuf:protobuf-java` (3.24.1)

## Usage

### Adding as Dependency

```xml
<dependency>
    <groupId>com.bharatml</groupId>
    <artifactId>feature-store-core</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Creating Feature Events

```java
import com.bharatml.featurestore.core.*;

FeatureEvent event = new FeatureEvent();
event.setEntityLabel("user");
event.setKeysSchema(Arrays.asList("user_id"));
Map<String, Object> keys = new HashMap<>();
keys.put("user_id", "12345");
event.setKeys(keys);
event.setFeatureGroupLabel("user_features");
event.setFeatureLabels(Arrays.asList("age", "score"));
Map<String, Object> features = new HashMap<>();
features.put("age", 30);
features.put("score", 95.5);
event.setFeatureValues(features);
```

### Converting to Protobuf

```java
import persist.QueryOuterClass;

// Convert to Protobuf Query
QueryOuterClass.Query query = FeatureConverter.toProto(event);

// Serialize to bytes
byte[] protobufBytes = query.toByteArray();
```

## Testing

Run unit tests:

```bash
mvn test
```

The module includes `FeatureConverterTest` which verifies Protobuf serialization.

## Protobuf Schema

The module uses `proto/feature.proto` from the parent `java-sdk/proto/` directory. The protobuf plugin generates Java classes during compilation.

Generated classes are in: `target/generated-sources/protobuf/java/persist/`

## Publishing

This module can be published to Maven repositories as a standalone artifact since it has no external framework dependencies.

