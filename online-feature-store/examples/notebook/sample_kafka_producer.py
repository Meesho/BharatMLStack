#!/usr/bin/env python3
"""
Kafka Feature Producer for Online Feature Store (Protobuf Version)
Publishes feature data as protobuf messages to the feature ingestion topic.
"""

import sys
import argparse
import json
from kafka import KafkaProducer
from kafka.errors import KafkaError

# Import the generated protobuf classes
# Make sure py-sdk is in your PYTHONPATH or install bharatml_commons
try:
    from bharatml_commons.proto.persist import persist_pb2
except ImportError:
    print("❌ Error: bharatml_commons package not found!")
    print("Install it with: pip install -e ../../py-sdk/bharatml_commons")
    sys.exit(1)


# Sample feature data
SAMPLE_DATA = {
  "entity_label": "test",
  "keys_schema": ["test_id"],
  "feature_group_schema": [
    {
      "label": "test_fg",
      "feature_labels": ["test_feature"]
    },
    {
      "label": "test_fg_fp32",
      "feature_labels": ["test_feature1", "test_feature2"]
    }
  ],
  "data": [
    {
      "key_values": ["1"],
      "feature_values": [
        {
          "values": {
            "bool_values": [False]
          }
        },
        {
          "values": {
            "fp32_values": [0.010070363990962505, 0.000014061562978895381]
          }
        }
      ]
    }
  ]
}


def json_to_protobuf(json_data):
    """Convert JSON feature data to protobuf Query message."""
    query = persist_pb2.Query()
    
    # Set entity_label
    query.entity_label = json_data["entity_label"]
    
    # Set keys_schema
    query.keys_schema.extend(json_data["keys_schema"])
    
    # Set feature_group_schema
    for fg_schema in json_data["feature_group_schema"]:
        fg = query.feature_group_schema.add()
        fg.label = fg_schema["label"]
        fg.feature_labels.extend(fg_schema["feature_labels"])
    
    # Set data
    for data_item in json_data["data"]:
        data = query.data.add()
        data.key_values.extend(data_item["key_values"])
        
        # Set feature_values
        for fv in data_item["feature_values"]:
            feature_value = data.feature_values.add()
            values = fv["values"]
            
            # Set the appropriate value type
            if "fp32_values" in values:
                feature_value.values.fp32_values.extend(values["fp32_values"])
            elif "fp64_values" in values:
                feature_value.values.fp64_values.extend(values["fp64_values"])
            elif "int32_values" in values:
                feature_value.values.int32_values.extend(values["int32_values"])
            elif "int64_values" in values:
                feature_value.values.int64_values.extend(values["int64_values"])
            elif "uint32_values" in values:
                feature_value.values.uint32_values.extend(values["uint32_values"])
            elif "uint64_values" in values:
                feature_value.values.uint64_values.extend(values["uint64_values"])
            elif "string_values" in values:
                feature_value.values.string_values.extend(values["string_values"])
            elif "bool_values" in values:
                feature_value.values.bool_values.extend(values["bool_values"])
            elif "vector" in values:
                for vec in values["vector"]:
                    vector = feature_value.values.vector.add()
                    vec_values = vec["values"]
                    if "fp32_values" in vec_values:
                        vector.values.fp32_values.extend(vec_values["fp32_values"])
                    # Add other vector types as needed
    
    return query


def create_producer(bootstrap_servers='localhost:9092'):
    """Create and return a Kafka producer instance for protobuf messages."""
    try:
        producer = KafkaProducer(
            bootstrap_servers=[bootstrap_servers],
            # Serialize protobuf to bytes
            value_serializer=lambda v: v.SerializeToString(),
            acks='all',
            retries=3,
            max_in_flight_requests_per_connection=1
        )
        print(f"✅ Connected to Kafka broker at {bootstrap_servers}")
        return producer
    except Exception as e:
        print(f"❌ Failed to create Kafka producer: {e}")
        sys.exit(1)


def send_message(producer, topic, message):
    """Send a protobuf message to the Kafka topic."""
    try:
        # message should be a protobuf Query object
        future = producer.send(topic, value=message)
        record_metadata = future.get(timeout=10)
        
        print(f"✅ Message sent successfully!")
        print(f"   Topic: {record_metadata.topic}")
        print(f"   Partition: {record_metadata.partition}")
        print(f"   Offset: {record_metadata.offset}")
        print(f"   Size: {len(message.SerializeToString())} bytes")
        return True
    except KafkaError as e:
        print(f"❌ Failed to send message: {e}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False


def load_json_file(filepath):
    """Load feature data from a JSON file."""
    try:
        with open(filepath, 'r') as f:
            data = json.load(f)
        print(f"✅ Loaded data from {filepath}")
        return data
    except FileNotFoundError:
        print(f"❌ File not found: {filepath}")
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"❌ Invalid JSON in file: {e}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description='Send feature data as Protobuf to Kafka topic for Online Feature Store'
    )
    parser.add_argument(
        '--broker',
        default='localhost:9092',
        help='Kafka bootstrap servers (default: localhost:9092)'
    )
    parser.add_argument(
        '--topic',
        default='online-feature-store.feature_ingestion',
        help='Kafka topic name (default: online-feature-store.feature_ingestion)'
    )
    parser.add_argument(
        '--file',
        help='Path to JSON file containing feature data'
    )
    parser.add_argument(
        '--sample',
        action='store_true',
        help='Send sample data (sub_order example)'
    )
    parser.add_argument(
        '--count',
        type=int,
        default=1,
        help='Number of messages to send (default: 1)'
    )
    parser.add_argument(
        '--show-json',
        action='store_true',
        help='Show the JSON data before converting to protobuf'
    )
    
    args = parser.parse_args()
    
    # Determine what data to send
    if args.file:
        json_data = load_json_file(args.file)
    elif args.sample:
        json_data = SAMPLE_DATA
        print("📤 Using sample sub_order data")
    else:
        print("❌ Please specify either --file or --sample")
        parser.print_help()
        sys.exit(1)
    
    # Display the JSON data if requested
    if args.show_json:
        print("\n📋 JSON Feature Data:")
        print(json.dumps(json_data, indent=2))
        print()
    
    # Convert JSON to protobuf
    print("🔄 Converting JSON to Protobuf...")
    proto_message = json_to_protobuf(json_data)
    print(f"✅ Converted to Protobuf Query message ({len(proto_message.SerializeToString())} bytes)")
    
    if args.show_json:
        print("\n📦 Protobuf Message:")
        print(proto_message)
        print()
    
    # Create producer
    producer = create_producer(args.broker)
    
    # Send message(s)
    success_count = 0
    for i in range(args.count):
        if args.count > 1:
            print(f"\n📤 Sending message {i+1}/{args.count}...")
        
        if send_message(producer, args.topic, proto_message):
            success_count += 1
    
    # Close producer
    producer.flush()
    producer.close()
    
    print(f"\n✨ Summary: {success_count}/{args.count} protobuf messages sent successfully")


if __name__ == "__main__":
    main()