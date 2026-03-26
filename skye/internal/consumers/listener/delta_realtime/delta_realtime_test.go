package delta_realtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractRateLimiterKey_Valid(t *testing.T) {
	r := &RealTimeDeltaConsumer{}

	// key format: /prefix/path/…/entity/…/model/…/variant/…/rate-limit
	// parts[4]=entity, parts[6]=model, parts[8]=variant
	key := "/skye/config/entity/product/models/reco_model/variants/v1/rate-limit"
	entity, model, variant := r.extractRateLimiterKey(key)
	assert.Equal(t, "product", entity)
	assert.Equal(t, "reco_model", model)
	assert.Equal(t, "v1", variant)
}

func TestExtractRateLimiterKey_BurstLimit(t *testing.T) {
	r := &RealTimeDeltaConsumer{}
	key := "/skye/config/entity/product/models/reco_model/variants/v1/burst-limit"
	entity, model, variant := r.extractRateLimiterKey(key)
	assert.Equal(t, "product", entity)
	assert.Equal(t, "reco_model", model)
	assert.Equal(t, "v1", variant)
}

func TestExtractRateLimiterKey_NoMatch(t *testing.T) {
	r := &RealTimeDeltaConsumer{}
	key := "/skye/config/entity/product/models/reco_model/variants/v1/something-else"
	entity, model, variant := r.extractRateLimiterKey(key)
	assert.Empty(t, entity)
	assert.Empty(t, model)
	assert.Empty(t, variant)
}
