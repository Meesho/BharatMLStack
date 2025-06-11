#!/bin/bash

set -e

echo "🗄️ BharatML Stack Database Initialization Starting..."
echo "=================================================="

# Wait for all services to be ready
echo "🔍 Step 1: Waiting for infrastructure services..."
./wait-for-services.sh

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

# Verify initialization
echo ""
echo "✅ Step 5: Verifying initialization..."
./verify-init.sh

echo ""
echo "🎉 Database initialization completed successfully!"
echo "==================================================" 