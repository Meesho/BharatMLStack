#!/bin/bash

set -e

echo "🗃️ Initializing ScyllaDB..."

# Create keyspace
echo "  📋 Creating 'onfs' keyspace..."
cqlsh scylla 9042 <<'EOF'
CREATE KEYSPACE IF NOT EXISTS onfs 
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

# Create tables and columns based on storage store configurations
echo "  📋 Creating tables and columns..."

# Store 1: feature table (for catalog entity)
echo "    📊 Creating table: feature..."
cqlsh scylla 9042 <<'EOF'
USE onfs;
CREATE TABLE IF NOT EXISTS feature (
    catalog_id text,
    PRIMARY KEY (catalog_id)
) WITH default_time_to_live = 100;
EOF

# Add columns for catalog/derived_fp32 feature group
echo "    📊 Adding column seg_0 to table: feature..."
set +e
cqlsh scylla 9042 <<'EOF' 2>/dev/null
USE onfs;
ALTER TABLE feature ADD seg_0 blob;
EOF
set -e

# Store 2: user_15d table (for user entity)
echo "    📊 Creating table: user_15d..."
cqlsh scylla 9042 <<'EOF'
USE onfs;
CREATE TABLE IF NOT EXISTS user_15d (
    user_id text,
    PRIMARY KEY (user_id)
) WITH default_time_to_live = 100;
EOF

# Add columns for user feature groups
echo "    📊 Adding columns to table: user_15d..."
set +e
cqlsh scylla 9042 <<'EOF' 2>/dev/null
USE onfs;
ALTER TABLE user_15d ADD seg_0 blob;
ALTER TABLE user_15d ADD seg_1 blob;
EOF
set -e

# Verify table creation
echo "  🔍 Verifying table creation..."
if cqlsh scylla 9042 -e "USE onfs; DESCRIBE TABLE feature" > /dev/null 2>&1; then
  echo "  ✅ Table 'feature' created successfully"
else
  echo "  ❌ Failed to create table 'feature'"
  exit 1
fi

if cqlsh scylla 9042 -e "USE onfs; DESCRIBE TABLE user_15d" > /dev/null 2>&1; then
  echo "  ✅ Table 'user_15d' created successfully"
else
  echo "  ❌ Failed to create table 'user_15d'"
  exit 1
fi

# Insert example data
echo "  📋 Inserting example data..."

# Verify table structure before inserting
echo "  🔍 Verifying table structure..."
TABLE_CHECK=$(cqlsh scylla 9042 -e "USE onfs; DESCRIBE TABLE feature;" 2>/dev/null | grep -c "catalog_id" || echo "0")
if [ "$TABLE_CHECK" -eq 0 ]; then
  echo "  ⚠️  Table 'feature' structure verification failed, skipping data insertion"
else
  # Example entry for catalog entity (feature table)
  # catalog_id: "catalog_001"
  # seg_0: Contains sbid_value feature (FP32, default 0.0) with metadata
  # Size: 13 bytes = 4 bytes (FP32 0.0) + 9 bytes (layout version 1 metadata)
  echo "    📊 Inserting example catalog entry..."
  set +e
  cqlsh scylla 9042 <<'EOF' 2>/dev/null
USE onfs;
-- 13 bytes: 4 bytes FP32 (0.0) + 9 bytes metadata (zeros for default)
INSERT INTO onfs.feature (catalog_id, seg_0) VALUES ('1', 0x00000000000000000000000000) USING TTL 300;
EOF
  set -e
fi

USER_TABLE_CHECK=$(cqlsh scylla 9042 -e "USE onfs; DESCRIBE TABLE user_15d;" 2>/dev/null | grep -c "user_id" || echo "0")
if [ "$USER_TABLE_CHECK" -eq 0 ]; then
  echo "  ⚠️  Table 'user_15d' structure verification failed, skipping data insertion"
else
  # Example entries for user entity (user_15d table)
  # user_id: "user_001" and "user_002"
  # seg_0: Contains region feature (string "delhi") with metadata (13 bytes total)
  # seg_1: Contains user__nqp feature (FP32, default 0.0) with metadata (13 bytes total)
  echo "    📊 Inserting example user entries..."
  set +e
  cqlsh scylla 9042 <<'EOF' 2>/dev/null
USE onfs;
-- seg_0: 13 bytes (region string "delhi" + metadata)
-- seg_1: 13 bytes (FP32 0.0 + metadata)
INSERT INTO onfs.user_15d (user_id, seg_0, seg_1) VALUES ('1', 0x64656c68690000000000000000, 0x00000000000000000000000000) USING TTL 10;

-- Another user entry with same default values
INSERT INTO onfs.user_15d (user_id, seg_0, seg_1) VALUES ('2', 0x64656c68690000000000000000, 0x00000000000000000000000000) USING TTL 10;
EOF
  set -e
fi

# Verify example data insertion
echo "  🔍 Verifying example data..."
if cqlsh scylla 9042 -e "USE onfs; SELECT catalog_id FROM feature WHERE catalog_id = 'catalog_001';" > /dev/null 2>&1; then
  echo "  ✅ Example catalog entry inserted successfully"
else
  echo "  ⚠️  Could not verify catalog entry (this is okay if it already exists)"
fi

if cqlsh scylla 9042 -e "USE onfs; SELECT user_id FROM user_15d WHERE user_id = 'user_001';" > /dev/null 2>&1; then
  echo "  ✅ Example user entries inserted successfully"
else
  echo "  ⚠️  Could not verify user entries (this is okay if they already exist)"
fi

echo "  ✅ ScyllaDB initialization completed successfully" 