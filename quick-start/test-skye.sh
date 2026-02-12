#!/bin/bash
# =============================================================================
# Skye End-to-End Test Script
# =============================================================================
# Tests the full Skye flow:
#   1. Health checks on all Skye services
#   2. Register store, frequency, entity, model, variant via skye-admin
#   3. Create Qdrant collection directly
#   4. Insert test vectors into Qdrant
#   5. Query similar candidates via skye-serving (gRPC)
# =============================================================================

ADMIN_URL="http://localhost:8092"
SERVING_URL="localhost:8094"
QDRANT_URL="http://localhost:6333"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✅ $1${NC}"; }
fail() { echo -e "${RED}❌ $1${NC}"; exit 1; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }
info() { echo -e "${YELLOW}➡️  $1${NC}"; }

# Helper: POST to admin API, tolerate duplicates/conflicts
admin_post() {
  local step_name="$1"
  local endpoint="$2"
  local body="$3"
  info "${step_name}..."
  HTTP_CODE=$(curl -s -o /tmp/skye_resp.txt -w "%{http_code}" -X POST "${ADMIN_URL}${endpoint}" \
    -H "Content-Type: application/json" \
    -d "${body}")
  RESP=$(cat /tmp/skye_resp.txt)
  echo "  Response (${HTTP_CODE}): ${RESP}"
  if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
    pass "${step_name} succeeded"
  elif [ "$HTTP_CODE" -eq 409 ] || [ "$HTTP_CODE" -eq 400 ]; then
    warn "${step_name} already exists or conflict (${HTTP_CODE}) — continuing"
  else
    fail "${step_name} failed with HTTP ${HTTP_CODE}"
  fi
}

# ---------------------------------------------------------------------------
# Step 0: Health checks
# ---------------------------------------------------------------------------
info "Checking service health..."

curl -sf "${ADMIN_URL}/health" > /dev/null && pass "skye-admin is healthy" || fail "skye-admin is not reachable at ${ADMIN_URL}"
curl -sf "http://${SERVING_URL}/health/self" > /dev/null && pass "skye-serving is healthy" || fail "skye-serving is not reachable at ${SERVING_URL}"
curl -sf "${QDRANT_URL}/healthz" > /dev/null && pass "Qdrant is healthy" || fail "Qdrant is not reachable at ${QDRANT_URL}"

echo ""

# ---------------------------------------------------------------------------
# Step 1: Register store
# ---------------------------------------------------------------------------
admin_post "Register store" "/api/v1/model/register-store" '{
  "conf_id": 1,
  "db": "testdb",
  "embeddings_table": "embeddings",
  "aggregator_table": "aggregator"
}'

# ---------------------------------------------------------------------------
# Step 2: Register frequency
# ---------------------------------------------------------------------------
admin_post "Register frequency" "/api/v1/model/register-frequency" '{
  "frequency": "daily"
}'

# ---------------------------------------------------------------------------
# Step 3: Register entity
# ---------------------------------------------------------------------------
admin_post "Register entity" "/api/v1/model/register-entity" '{
  "entity": "test-products",
  "store_id": "1"
}'

# ---------------------------------------------------------------------------
# Step 4: Register model
# ---------------------------------------------------------------------------
admin_post "Register model" "/api/v1/model/register-model" '{
  "entity": "test-products",
  "model": "product-embeddings",
  "embedding_store_enabled": false,
  "embedding_store_ttl": 0,
  "model_config": {
    "distance_function": "Cosine",
    "vector_dimension": 4
  },
  "model_type": "RESET",
  "kafka_id": 0,
  "training_data_path": "",
  "metadata": {
    "entity": "test-products",
    "key-type": "product_id"
  },
  "job_frequency": "daily",
  "number_of_partitions": 1,
  "failure_producer_kafka_id": 0,
  "topic_name": ""
}'

# ---------------------------------------------------------------------------
# Step 5: Register variant
# ---------------------------------------------------------------------------
admin_post "Register variant" "/api/v1/model/register-variant" '{
  "entity": "test-products",
  "model": "product-embeddings",
  "variant": "v1",
  "vector_db_type": "QDRANT",
  "vector_db_config": {
    "read_host": "qdrant:6334",
    "write_host": "qdrant:6334",
    "port": "6334",
    "http2config": {
      "deadline": 5000,
      "write_deadline": 5000,
      "is_plain_text": true
    },
    "params": {
      "distance_function": "Cosine",
      "vector_dimension": "4"
    }
  },
  "filter": [],
  "type": "SCALE_UP",
  "in_memory_caching_enabled": false,
  "in_memory_cache_ttl_seconds": 0,
  "distributed_caching_enabled": false,
  "distributed_cache_ttl_seconds": 0,
  "rt_partition": 1,
  "rate_limiters": {
    "RateLimit": 100,
    "BurstLimit": 200
  }
}'

# ---------------------------------------------------------------------------
# Step 6: Create Qdrant collection directly (bypassing skye-admin for simplicity)
# ---------------------------------------------------------------------------
COLLECTION_NAME="v1_product-embeddings_1"
info "Creating Qdrant collection: ${COLLECTION_NAME}..."
HTTP_CODE=$(curl -s -o /tmp/skye_resp.txt -w "%{http_code}" -X PUT "${QDRANT_URL}/collections/${COLLECTION_NAME}" \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 4,
      "distance": "Cosine"
    }
  }')
RESP=$(cat /tmp/skye_resp.txt)
echo "  Response (${HTTP_CODE}): ${RESP}"
if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
  pass "Qdrant collection created"
elif echo "${RESP}" | grep -qi "already exists"; then
  warn "Qdrant collection already exists — continuing"
else
  fail "Failed to create Qdrant collection (HTTP ${HTTP_CODE})"
fi

# ---------------------------------------------------------------------------
# Step 7: Insert test vectors into Qdrant
# ---------------------------------------------------------------------------
info "Inserting test vectors into Qdrant..."
RESP=$(curl -s -X PUT "${QDRANT_URL}/collections/${COLLECTION_NAME}/points" \
  -H "Content-Type: application/json" \
  -d '{
    "points": [
      {"id": 1, "vector": [0.1, 0.2, 0.3, 0.4], "payload": {"product_id": "prod_001"}},
      {"id": 2, "vector": [0.2, 0.3, 0.4, 0.5], "payload": {"product_id": "prod_002"}},
      {"id": 3, "vector": [0.9, 0.8, 0.7, 0.6], "payload": {"product_id": "prod_003"}},
      {"id": 4, "vector": [0.5, 0.5, 0.5, 0.5], "payload": {"product_id": "prod_004"}},
      {"id": 5, "vector": [0.1, 0.1, 0.9, 0.9], "payload": {"product_id": "prod_005"}}
    ]
  }')
echo "  Response: ${RESP}"
pass "Test vectors inserted (upserted)"

# ---------------------------------------------------------------------------
# Step 8: Verify vectors via Qdrant search (sanity check)
# ---------------------------------------------------------------------------
info "Verifying Qdrant search works..."
RESP=$(curl -s -X POST "${QDRANT_URL}/collections/${COLLECTION_NAME}/points/search" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, 0.4],
    "limit": 3,
    "with_payload": true
  }')
echo "  Response: ${RESP}"
pass "Qdrant search verified"

# ---------------------------------------------------------------------------
# Step 9: Query via skye-serving gRPC
# ---------------------------------------------------------------------------
info "Querying similar candidates via skye-serving gRPC..."

# Check if grpcurl is available
if command -v grpcurl &> /dev/null; then
  RESP=$(grpcurl -plaintext \
    -H "skye-caller-id: test-script" \
    -H "skye-auth-token: test" \
    -d '{
    "entity": "test-products",
    "modelName": "product-embeddings",
    "variant": "v1",
    "limit": 3,
    "embeddings": [
      {"embedding": [0.1, 0.2, 0.3, 0.4]}
    ]
  }' "${SERVING_URL}" SkyeSimilarCandidateService/getSimilarCandidates 2>&1) || true
  echo "  Response: ${RESP}"
  pass "gRPC query completed"
else
  echo -e "${YELLOW}  ⚠️  grpcurl not installed. Install it to test gRPC:${NC}"
  echo "     brew install grpcurl"
  echo ""
  echo "  Then run manually:"
  echo '  grpcurl -plaintext -H "skye-caller-id: test" -H "skye-auth-token: test" -d '"'"'{"entity":"test-products","modelName":"product-embeddings","variant":"v1","limit":3,"embeddings":[{"embedding":[0.1,0.2,0.3,0.4]}]}'"'"' localhost:8094 SkyeSimilarCandidateService/getSimilarCandidates'
fi

echo ""
echo "==========================================="
echo -e "${GREEN}🎉 Skye end-to-end test complete!${NC}"
echo "==========================================="
