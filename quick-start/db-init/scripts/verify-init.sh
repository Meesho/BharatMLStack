#!/bin/bash

set -e

echo "✅ Verifying all database initialization..."

# Verify ScyllaDB
echo -n "  🗃️ ScyllaDB keyspace 'onfs'... "
if cqlsh scylla 9042 -e "DESCRIBE KEYSPACE onfs;" > /dev/null 2>&1; then
  echo "✅"
else
  echo "❌"
  exit 1
fi

# Verify MySQL
echo -n "  🗂️ MySQL admin user... "
ADMIN_COUNT=$(mysql -hmysql -uroot -proot testdb -sN -e "SELECT COUNT(*) FROM users WHERE email='admin@admin.com';" 2>/dev/null || echo "0")
if [ "$ADMIN_COUNT" -eq 1 ]; then
  echo "✅"
else
  echo "❌"
  exit 1
fi

# Verify etcd
echo -n "  🔧 etcd configuration key... "
if etcdctl --endpoints=http://etcd:2379 get /config/onfs > /dev/null 2>&1; then
  echo "✅"
else
  echo "❌"
  exit 1
fi

# Test Redis connectivity
echo -n "  🗄️ Redis connectivity... "
if redis-cli -h redis ping > /dev/null 2>&1; then
  echo "✅"
else
  echo "❌"
  exit 1
fi

echo ""
echo "🎉 All database services initialized and verified successfully!"
echo ""
echo "📋 Summary:"
echo "  • ScyllaDB: 'onfs' keyspace ready"
echo "  • MySQL: All tables created, admin user: admin@admin.com"
echo "  • etcd: Configuration keys set"
echo "  • Redis: Ready for caching" 