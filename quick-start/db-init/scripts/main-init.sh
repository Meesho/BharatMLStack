#!/bin/bash

set -e

echo "🗄️ BharatML Stack Database Initialization Starting..."
echo "=================================================="

# Wait for all services to be ready
echo "🔍 Step 1: Waiting for infrastructure services..."
./wait-for-services.sh

# Check if dummy data initialization is requested
if [[ "${INIT_DUMMY_DATA:-false}" == "true" ]]; then
  echo ""
  echo "📦 Dummy data initialization mode enabled"
  
  # Initialize ScyllaDB (dummy script includes keyspace creation)
  echo ""
  echo "🗃️ Step 2: Initializing ScyllaDB with dummy data..."
  if [ -f ./init-dummy-scylla.sh ]; then
    ./init-dummy-scylla.sh
  else
    echo "  ⏭️  Skipping ScyllaDB dummy data (init-dummy-scylla.sh not found), running regular init..."
    ./init-scylla.sh
  fi
  
  # Initialize MySQL (regular first to create tables, then dummy to insert data)
  echo ""
  echo "🗂️ Step 3: Initializing MySQL..."
  ./init-mysql.sh
  
  echo ""
  echo "🗂️ Step 3b: Inserting dummy data into MySQL..."
  if [ -f ./init-dummy-mysql.sh ]; then
    ./init-dummy-mysql.sh
  else
    echo "  ⏭️  Skipping MySQL dummy data (init-dummy-mysql.sh not found)"
  fi
  
  # Initialize etcd (dummy script includes basic setup)
  echo ""
  echo "🔧 Step 4: Initializing etcd with dummy data..."
  if [ -f ./init-dummy-etcd.sh ]; then
    ./init-dummy-etcd.sh
  else
    echo "  ⏭️  Skipping etcd dummy data (init-dummy-etcd.sh not found), running regular init..."
    ./init-etcd.sh
  fi
else
  # Regular initialization
  # Initialize ScyllaDB
  echo ""
  echo "🗃️ Step 2: Initializing ScyllaDB..."
  ./init-scylla.sh
  
  # Initialize MySQL
  echo ""
  echo "🗂️ Step 3: Initializing MySQL..."
  ./init-mysql.sh
  
  # Initialize etcd
  echo ""
  echo "🔧 Step 4: Initializing etcd..."
  ./init-etcd.sh
fi

# Verify initialization
echo ""
echo "✅ Step 5: Verifying initialization..."
./verify-init.sh

echo ""
echo "🎉 Database initialization completed successfully!"
echo "==================================================" 