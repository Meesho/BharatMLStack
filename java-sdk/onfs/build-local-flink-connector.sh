#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

JAVA_VERSION="${JAVA_VERSION:-11}"

echo "=========================================="
echo "Building Feature Store Flink Connector"
echo "Java version: $JAVA_VERSION"
echo "=========================================="

# Build all modules using parent POM
mvn -Djava.version=$JAVA_VERSION clean package

echo ""
echo "=========================================="
echo "Build completed successfully!"
echo "=========================================="
echo ""
echo "Artifact JARs:"
echo "  Core: feature-store-core/target/feature-store-core-1.0.0.jar"
echo "  Kafka Client: feature-store-kafka-client/target/feature-store-kafka-client-1.0.0.jar"
echo "  Connector: feature-store-flink-connector-sdk-flink1x/target/feature-store-flink-connector-sdk-flink1x-1.0.0.jar"
echo ""
echo "To use the connector in a Flink job:"
echo ""
echo "1. Include connector jar in Flink job classpath:"
echo "   - Copy to \$FLINK_HOME/lib/ (for all jobs), or"
echo "   - Include in job fat jar via Maven dependency"
echo ""
echo "2. Example Flink job dependency (pom.xml):"
echo "   <dependency>"
echo "     <groupId>com.bharatml</groupId>"
echo "     <artifactId>feature-store-flink-connector-sdk-flink1x</artifactId>"
echo "     <version>1.0.0</version>"
echo "   </dependency>"
echo ""
echo "3. Example usage in Flink job:"
echo "   FeatureStoreClientConfig config = FeatureStoreClientConfig.builder()"
echo "       .bootstrapServers(\"localhost:9092\")"
echo "       .topic(\"my-topic\")"
echo "       .build();"
echo "   events.addSink(new FeatureStoreKafkaSink(config));"
echo ""

