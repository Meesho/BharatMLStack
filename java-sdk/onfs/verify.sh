#!/bin/bash

# Script to verify Protobuf messages in Kafka topic
# Requires: kafka-console-consumer and protoc

set -e

BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
TOPIC="${KAFKA_TOPIC:-online-feature-store.feature_ingestion}"
MAX_MESSAGES="${MAX_MESSAGES:-10}"

echo "=========================================="
echo "Verifying Protobuf messages in Kafka"
echo "=========================================="
echo "Bootstrap servers: $BOOTSTRAP_SERVERS"
echo "Topic: $TOPIC"
echo "Max messages: $MAX_MESSAGES"
echo ""

# Check if topic exists
echo "Checking if topic exists..."
kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --describe --topic "$TOPIC" || {
    echo "ERROR: Topic '$TOPIC' does not exist or Kafka is not accessible"
    echo "Create the topic first:"
    echo "  kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVERS --create --topic $TOPIC --partitions 1 --replication-factor 1"
    exit 1
}

echo ""
echo "=========================================="
echo "Decoding Protobuf Messages"
echo "=========================================="
echo ""
echo "To decode Protobuf messages, use protoc:"
echo ""
echo "  # Consume raw bytes and decode"
echo "  kafka-console-consumer.sh \\"
echo "    --bootstrap-server $BOOTSTRAP_SERVERS \\"
echo "    --topic $TOPIC \\"
echo "    --from-beginning \\"
echo "    --max-messages $MAX_MESSAGES \\"
echo "    --property print.key=true \\"
echo "    --property print.timestamp=true | \\"
echo "  while IFS=\$'\\t' read -r key timestamp value; do"
echo "    echo \"Key: \$key, Timestamp: \$timestamp\""
echo "    echo -n \"\$value\" | base64 -d | \\"
echo "      protoc --decode=persist.Query proto/feature.proto"
echo "    echo \"---\""
echo "  done"
echo ""
echo "Or use protoc --decode_raw for raw wire format:"
echo ""
echo "  kafka-console-consumer.sh \\"
echo "    --bootstrap-server $BOOTSTRAP_SERVERS \\"
echo "    --topic $TOPIC \\"
echo "    --from-beginning \\"
echo "    --max-messages 1 | \\"
echo "  tail -1 | cut -f3 | base64 -d | protoc --decode_raw"
echo ""
echo "=========================================="
echo "Sample consumption (first message, raw bytes):"
echo "=========================================="

kafka-console-consumer.sh \
    --bootstrap-server "$BOOTSTRAP_SERVERS" \
    --topic "$TOPIC" \
    --from-beginning \
    --max-messages 1 \
    --property print.key=true \
    --property print.value=false \
    --property print.timestamp=true || {
    echo "Note: Raw Protobuf bytes are not human-readable."
    echo "Use protoc to decode messages as shown above."
}

echo ""
echo "=========================================="
echo "Verification complete"
echo "=========================================="

