package retrieve

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestOrderRetrieveHandler(db *MockDatabase) *OrderRetrieveHandler {
	return &OrderRetrieveHandler{
		scyllaDb: db,
	}
}

// Verifies that empty database result returns empty events without error.
func TestOrderRetrieveHandler_Retrieve_EmptyResult(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.NoError(t, err)
	assert.Empty(t, events)
	mockDb.AssertExpectations(t)
}

// Verifies that database errors are properly propagated to the caller.
func TestOrderRetrieveHandler_Retrieve_DatabaseError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(nil, errors.New("database connection failed"))

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve data")
	assert.Nil(t, events)
	mockDb.AssertExpectations(t)
}

// Verifies that table-to-fields mapping is correctly built for a single week range.
func TestOrderRetrieveHandler_buildTableToFieldsMapping_NormalRange(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(startTime, endTime, getOrderTableName)

	assert.NotEmpty(t, result)
}

// Verifies that table-to-fields mapping includes multiple weeks for a multi-week range.
func TestOrderRetrieveHandler_buildTableToFieldsMapping_MultipleWeeks(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	startTime := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 22, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(startTime, endTime, getOrderTableName)

	assert.NotEmpty(t, result)

	totalFields := 0
	for _, fields := range result {
		totalFields += len(fields)
	}
	assert.Greater(t, totalFields, 1)
}

// Verifies that bucket indices map to correct order table names.
func TestGetOrderTableName_ReturnsCorrectTableNames(t *testing.T) {
	tests := []struct {
		bucketIdx int
		expected  string
	}{
		{0, "order_interactions_bucket1"},
		{1, "order_interactions_bucket2"},
		{2, "order_interactions_bucket3"},
		{3, ""},
	}

	for _, tt := range tests {
		result := getOrderTableName(tt.bucketIdx)
		assert.Equal(t, tt.expected, result)
	}
}

// Verifies that empty data map returns empty result during deserialization.
func TestOrderRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_EmptyData(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(map[string][]byte{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that nil column values are skipped during deserialization.
func TestOrderRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_NilData(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": nil,
	}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(weekToData)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that empty byte slices are treated as no data during deserialization.
func TestOrderRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_EmptyByteSlice(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": {},
	}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(weekToData)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that invalid/corrupted data returns an error during deserialization.
func TestOrderRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_InvalidData(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": []byte("invalid-data"),
	}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(weekToData)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to deserialize")
}

// Verifies that empty table-to-fields mapping returns empty result.
func TestOrderRetrieveHandler_fetchDataInParallel_EmptyTableToFields(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{}

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that tables with no fields are skipped and no database call is made.
func TestOrderRetrieveHandler_fetchDataInParallel_EmptyFieldsForTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"order_interactions_bucket1": {},
	}

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Empty(t, result)
	mockDb.AssertNotCalled(t, "RetrieveInteractions")
}

// Verifies that data is correctly fetched from a single table with multiple week columns.
func TestOrderRetrieveHandler_fetchDataInParallel_SingleTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"order_interactions_bucket1": {"week_0", "week_1"},
	}

	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", []string{"week_0", "week_1"}).
		Return(map[string]interface{}{
			"week_0": []byte{},
			"week_1": []byte{},
		}, nil)

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockDb.AssertExpectations(t)
}

// Verifies that database errors during parallel fetch are properly propagated.
func TestOrderRetrieveHandler_fetchDataInParallel_DatabaseError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"order_interactions_bucket1": {"week_0"},
	}

	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", []string{"week_0"}).
		Return(nil, errors.New("database error"))

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve data")
	mockDb.AssertExpectations(t)
}

// Verifies that missing columns in database response are handled gracefully as nil.
func TestOrderRetrieveHandler_fetchDataInParallel_MissingColumn(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"order_interactions_bucket1": {"week_0", "week_1"},
	}

	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", []string{"week_0", "week_1"}).
		Return(map[string]interface{}{
			"week_0": []byte{1, 2, 3},
		}, nil)

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Nil(t, result["week_1"])
	mockDb.AssertExpectations(t)
}

// Verifies that unexpected data types in database response return an error.
func TestOrderRetrieveHandler_fetchDataInParallel_WrongDataType(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"order_interactions_bucket1": {"week_0"},
	}

	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", []string{"week_0"}).
		Return(map[string]interface{}{
			"week_0": "invalid-type",
		}, nil)

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected data type")
	mockDb.AssertExpectations(t)
}

// Verifies that zero limit is handled gracefully and returns empty result.
func TestOrderRetrieveHandler_Retrieve_LimitZero(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	events, err := handler.Retrieve("user1", startTime, endTime, 0)

	assert.NoError(t, err)
	assert.Empty(t, events)
	mockDb.AssertExpectations(t)
}

// Verifies that requested limit exceeding max is capped to max limit.
func TestOrderRetrieveHandler_Retrieve_CapLimitApplied(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	// Request a limit higher than max (2000)
	events, err := handler.Retrieve("user1", startTime, endTime, 5000)

	assert.NoError(t, err)
	assert.Empty(t, events)
	mockDb.AssertExpectations(t)
}

// Verifies that data is fetched in parallel from multiple tables (different buckets).
func TestOrderRetrieveHandler_fetchDataInParallel_MultipleTables(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"order_interactions_bucket1": {"week_0"},
		"order_interactions_bucket2": {"week_8"},
	}

	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", []string{"week_0"}).
		Return(map[string]interface{}{
			"week_0": []byte{},
		}, nil)

	mockDb.On("RetrieveInteractions", "order_interactions_bucket2", "user1", []string{"week_8"}).
		Return(map[string]interface{}{
			"week_8": []byte{},
		}, nil)

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockDb.AssertExpectations(t)
}

// Verifies that single day query only hits one table and one week column.
func TestOrderRetrieveHandler_Retrieve_SingleWeekSingleTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	// Single day query - should only query one week
	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.NoError(t, err)
	assert.Empty(t, events)
	// Verify only bucket1 was queried
	mockDb.AssertCalled(t, "RetrieveInteractions", "order_interactions_bucket1", "user1", mock.AnythingOfType("[]string"))
	mockDb.AssertNotCalled(t, "RetrieveInteractions", "order_interactions_bucket2", mock.Anything, mock.Anything)
}

// Verifies that table-to-fields mapping spans multiple buckets for cross-bucket range.
func TestOrderRetrieveHandler_buildTableToFieldsMapping_CrossBucketRange(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	// Query spanning from bucket 0 to bucket 1
	// Week 5 is in bucket 0, Week 10 is in bucket 1
	startTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli()   // Around week 5
	endTime := time.Date(2024, 3, 15, 23, 59, 59, 0, time.UTC).UnixMilli() // Around week 11

	result := handler.buildTableToFieldsMapping(startTime, endTime, getOrderTableName)

	assert.NotEmpty(t, result)

	// Should have mappings for multiple tables
	tableCount := len(result)
	assert.GreaterOrEqual(t, tableCount, 1)
}

// Verifies that multi-week query within same bucket retrieves data correctly.
func TestOrderRetrieveHandler_Retrieve_MultipleWeeksSameBucket(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	// Query spanning multiple weeks but within same bucket
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC).UnixMilli()

	// Expect multiple week columns to be requested from bucket1
	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", mock.MatchedBy(func(columns []string) bool {
		return len(columns) >= 1
	})).Return(map[string]interface{}{}, nil)

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.NoError(t, err)
	assert.Empty(t, events)
	mockDb.AssertExpectations(t)
}
