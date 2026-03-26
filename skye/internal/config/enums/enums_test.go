package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelTypeConstants(t *testing.T) {
	assert.Equal(t, ModelType("RESET"), RESET)
	assert.Equal(t, ModelType("DELTA"), DELTA)
}

func TestVariantStateConstants(t *testing.T) {
	assert.Equal(t, VariantState("DATA_INGESTION_STARTED"), DATA_INGESTION_STARTED)
	assert.Equal(t, VariantState("DATA_INGESTION_COMPLETED"), DATA_INGESTION_COMPLETED)
	assert.Equal(t, VariantState("INDEXING_STARTED"), INDEXING_STARTED)
	assert.Equal(t, VariantState("INDEXING_IN_PROGRESS"), INDEXING_IN_PROGRESS)
	assert.Equal(t, VariantState("INDEXING_COMPLETED_WITH_RESET"), INDEXING_COMPLETED_WITH_RESET)
	assert.Equal(t, VariantState("INDEXING_COMPLETED"), INDEXING_COMPLETED)
	assert.Equal(t, VariantState("MODEL_VERSION_UPDATED"), MODEL_VERSION_UPDATED)
	assert.Equal(t, VariantState("COMPLETED"), COMPLETED)
}

func TestVariantTypeConstants(t *testing.T) {
	assert.Equal(t, Type("SCALE_UP"), SCALE_UP)
	assert.Equal(t, Type("EXPERIMENT"), EXPERIMENT)
}

func TestVectorDbTypeConstants(t *testing.T) {
	assert.Equal(t, VectorDbType("QDRANT"), QDRANT)
	assert.Equal(t, VectorDbType("NGT"), NGT)
	assert.Equal(t, VectorDbType("EIGENIX"), EIGENIX)
}
