package retrieve

import (
	"errors"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
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

	result := handler.buildTableToFieldsMapping(handler.buildOrderedWeeks(startTime, endTime))

	assert.NotEmpty(t, result)
}

// Verifies that table-to-fields mapping includes multiple weeks for a multi-week range.
func TestOrderRetrieveHandler_buildTableToFieldsMapping_MultipleWeeks(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	startTime := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 22, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(handler.buildOrderedWeeks(startTime, endTime))

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

// Verifies that zero limit returns empty result without calling the DB.
func TestOrderRetrieveHandler_Retrieve_LimitZero(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, 0)

	assert.ErrorIs(t, err, ErrInvalidLimit)
	assert.Nil(t, events)
	mockDb.AssertExpectations(t)
}

// Verifies that negative limit returns ErrInvalidLimit without calling the DB.
func TestOrderRetrieveHandler_Retrieve_NegativeLimit(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, -1)

	assert.ErrorIs(t, err, ErrInvalidLimit)
	assert.Nil(t, events)
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

	startTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 3, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(handler.buildOrderedWeeks(startTime, endTime))

	assert.NotEmpty(t, result)
	assert.GreaterOrEqual(t, len(result), 1)
}

func TestOrderRetrieveHandler_Retrieve_TimeRangeExceeded(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	// Use a relative past window (>24 weeks) so we always get ErrTimeRangeExceeded, not ErrEndTimeInFuture
	startTime := time.Now().Add(-26 * 7 * 24 * time.Hour).UnixMilli()
	endTime := time.Now().Add(-1 * time.Hour).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.ErrorIs(t, err, ErrTimeRangeExceeded)
	assert.Nil(t, events)
}

func TestOrderRetrieveHandler_Retrieve_EndTimeInFuture(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	startTime := time.Now().Add(-1 * time.Hour).UnixMilli()
	endTime := time.Now().Add(24 * time.Hour).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.ErrorIs(t, err, ErrEndTimeInFuture)
	assert.Nil(t, events)
}

func TestOrderRetrieveHandler_buildOrderedWeeks_NormalRange(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	startTime := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC).UnixMilli()

	weeks := handler.buildOrderedWeeks(startTime, endTime)

	assert.NotEmpty(t, weeks)
	for i := 1; i < len(weeks); i++ {
		assert.Greater(t, weeks[i-1], weeks[i])
	}
}

func TestOrderRetrieveHandler_buildOrderedWeeks_SingleWeek(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	weeks := handler.buildOrderedWeeks(startTime, endTime)

	assert.Len(t, weeks, 1)
}

// TestOrderRetrieveHandler_buildOrderedWeeks_WrapAround covers the branch when
// lowerBound > upperBound (range crosses week-24 boundary).
func TestOrderRetrieveHandler_buildOrderedWeeks_WrapAround(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	startTime := time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC).UnixMilli()

	weeks := handler.buildOrderedWeeks(startTime, endTime)

	assert.NotEmpty(t, weeks)
	for _, w := range weeks {
		assert.GreaterOrEqual(t, w, 0)
		assert.Less(t, w, 24)
	}
}

func TestOrderRetrieveHandler_mergeFilterAndLimit_FiltersAndLimits(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	start := int64(100)
	end := int64(500)

	weekToEvents := map[string][]model.FlattenedOrderEvent{
		"week_3": {
			{OrderedAt: 450, CatalogID: 1},
			{OrderedAt: 300, CatalogID: 2},
			{OrderedAt: 200, CatalogID: 3},
			{OrderedAt: 50, CatalogID: 4},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, start, end, 10)

	assert.Len(t, result, 3)
	assert.Equal(t, int32(1), result[0].CatalogID)
	assert.Equal(t, int32(2), result[1].CatalogID)
	assert.Equal(t, int32(3), result[2].CatalogID)
}

func TestOrderRetrieveHandler_mergeFilterAndLimit_RespectsLimit(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToEvents := map[string][]model.FlattenedOrderEvent{
		"week_3": {
			{OrderedAt: 400},
			{OrderedAt: 300},
			{OrderedAt: 200},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, 100, 500, 2)

	assert.Len(t, result, 2)
}

func TestOrderRetrieveHandler_mergeFilterAndLimit_MultipleWeeks(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToEvents := map[string][]model.FlattenedOrderEvent{
		"week_4": {
			{OrderedAt: 900, CatalogID: 1},
			{OrderedAt: 800, CatalogID: 2},
		},
		"week_3": {
			{OrderedAt: 500, CatalogID: 3},
			{OrderedAt: 400, CatalogID: 4},
		},
	}

	result := handler.mergeFilterAndLimit([]int{4, 3}, weekToEvents, 100, 1000, 10)

	assert.Len(t, result, 4)
	assert.Equal(t, int64(900), result[0].OrderedAt)
	assert.Equal(t, int64(800), result[1].OrderedAt)
	assert.Equal(t, int64(500), result[2].OrderedAt)
	assert.Equal(t, int64(400), result[3].OrderedAt)
}

func TestOrderRetrieveHandler_mergeFilterAndLimit_EmptyWeeks(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToEvents := map[string][]model.FlattenedOrderEvent{}

	result := handler.mergeFilterAndLimit([]int{3, 4}, weekToEvents, 100, 500, 10)

	assert.Empty(t, result)
}

func TestOrderRetrieveHandler_mergeFilterAndLimit_AllEventsOutOfRange(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToEvents := map[string][]model.FlattenedOrderEvent{
		"week_3": {
			{OrderedAt: 900},
			{OrderedAt: 800},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, 100, 500, 10)

	assert.Empty(t, result)
}

func TestOrderRetrieveHandler_mergeFilterAndLimit_SkipsEventsAboveEnd(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToEvents := map[string][]model.FlattenedOrderEvent{
		"week_3": {
			{OrderedAt: 600},
			{OrderedAt: 400},
			{OrderedAt: 200},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, 100, 500, 10)

	assert.Len(t, result, 2)
	assert.Equal(t, int64(400), result[0].OrderedAt)
	assert.Equal(t, int64(200), result[1].OrderedAt)
}

func TestOrderRetrieveHandler_deserializeWeeks_EmptyData(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": nil,
		"week_1": {},
	}

	result, err := handler.deserializeWeeks(weekToData, "user1")

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestOrderRetrieveHandler_deserializeWeeks_InvalidData(t *testing.T) {
	handler := &OrderRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": []byte("corrupt"),
	}

	result, err := handler.deserializeWeeks(weekToData, "user1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to deserialize")
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
