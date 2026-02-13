#!/bin/bash

set -e

echo "🗃️ Initializing ScyllaDB..."

# Create keyspace
echo "  📋 Creating 'onfs' keyspace..."
cqlsh scylla 9042 <<'EOF'
CREATE KEYSPACE IF NOT EXISTS onfs 
    WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};
EOF

# Create keyspace
echo "  📋 Creating 'skye' keyspace..."
cqlsh scylla 9042 <<'EOF'
CREATE KEYSPACE IF NOT EXISTS skye 
    WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};
EOF

# Verify keyspace creation
echo "  🔍 Verifying keyspace creation..."
if cqlsh scylla 9042 -e "DESCRIBE KEYSPACE onfs" > /dev/null 2>&1; then
  echo "  ✅ ScyllaDB keyspace 'onfs' created successfully"
else
  echo "  ❌ Failed to create ScyllaDB keyspace"
  exit 1
fi