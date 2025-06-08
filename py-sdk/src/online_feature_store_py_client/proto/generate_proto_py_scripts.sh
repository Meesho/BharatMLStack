#!/bin/bash
set -euo pipefail  # exit on error or script failure

# Note: This script requires the protoc compiler to be installed. https://protobuf.dev/installation/ 

# Store current working directory
CURRENT_DIR=$(pwd)

# Base paths
PROTO_SRC_DIR="../../../../../pkg/proto"
PYTHON_OUT_BASE="../../../sdks/python/src/online-feature-store-py-client/proto"

# Array of proto files to process
PROTO_FILES=("persist" "retrieve")

for FILE in "${PROTO_FILES[@]}"; do
    SRC_PROTO="$PROTO_SRC_DIR/$FILE.proto"
    TARGET_PROTO_DIR="$PROTO_SRC_DIR/$FILE"
    PYTHON_OUT_DIR="$PYTHON_OUT_BASE/$FILE"

    # Create the target directory if it doesn't exist
    mkdir -p "$TARGET_PROTO_DIR"

    # Copy the .proto file into its own directory (to make protoc output clean)
    cp "$SRC_PROTO" "$TARGET_PROTO_DIR/"

    # Generate Python code from the .proto file
    (
        cd "$TARGET_PROTO_DIR"
        protoc -I. --python_out="$PYTHON_OUT_DIR" "$FILE.proto"
    )

    # Remove the copied .proto file
    rm "$TARGET_PROTO_DIR/$FILE.proto"
done

# Return to the original directory
cd "$CURRENT_DIR"
