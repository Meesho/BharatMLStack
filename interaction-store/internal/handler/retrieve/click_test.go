package retrieve

import (
	"errors"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	m.Run()
}

// MockDatabase is a mock implementation of the scylla.Database interface
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) RetrieveInteractions(tableName string, userId string, columns []string) (map[string]interface{}, error) {
	args := m.Called(tableName, userId, columns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockDatabase) PersistInteractions(tableName string, userId string, columns map[string]interface{}) error {
	args := m.Called(tableName, userId, columns)
	return args.Error(0)
}

func (m *MockDatabase) UpdateInteractions(tableName string, userId string, columns map[string]interface{}) error {
	args := m.Called(tableName, userId, columns)
	return args.Error(0)
}

func (m *MockDatabase) RetrieveMetadata(metadataTableName string, userId string, columns []string) (map[string]interface{}, error) {
	args := m.Called(metadataTableName, userId, columns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockDatabase) PersistMetadata(metadataTableName string, userId string, columns map[string]interface{}) error {
	args := m.Called(metadataTableName, userId, columns)
	return args.Error(0)
}

func (m *MockDatabase) UpdateMetadata(metadataTableName string, userId string, columns map[string]interface{}) error {
	args := m.Called(metadataTableName, userId, columns)
	return args.Error(0)
}

func newTestClickRetrieveHandler(db *MockDatabase) *ClickRetrieveHandler {
	return &ClickRetrieveHandler{
		scyllaDb: db,
	}
}

// Verifies that empty database result returns empty events without error.
func TestClickRetrieveHandler_Retrieve_EmptyResult(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

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
func TestClickRetrieveHandler_Retrieve_DatabaseError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

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

// Verifies that capLimit returns original limit when below maximum.
func TestCapLimit_ReturnsLimitWhenBelowMax(t *testing.T) {
	result := capLimit(100, 2000)
	assert.Equal(t, int32(100), result)
}

// Verifies that capLimit returns max limit when requested limit exceeds maximum.
func TestCapLimit_ReturnsMaxWhenAboveMax(t *testing.T) {
	result := capLimit(3000, 2000)
	assert.Equal(t, int32(2000), result)
}

// Verifies that capLimit returns max limit when equal to maximum.
func TestCapLimit_ReturnsMaxWhenEqual(t *testing.T) {
	result := capLimit(2000, 2000)
	assert.Equal(t, int32(2000), result)
}

// Verifies that table-to-fields mapping is correctly built for a single week range.
func TestClickRetrieveHandler_buildTableToFieldsMapping_NormalRange(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(handler.buildOrderedWeeks(startTime, endTime))

	assert.NotEmpty(t, result)
}

// Verifies that table-to-fields mapping includes multiple weeks for a multi-week range.
func TestClickRetrieveHandler_buildTableToFieldsMapping_MultipleWeeks(t *testing.T) {
	handler := &ClickRetrieveHandler{}

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

// Verifies that bucket indices map to correct click table names.
func TestGetTableName_ClickRetrieve_ReturnsCorrectTableNames(t *testing.T) {
	tests := []struct {
		bucketIdx int
		expected  string
	}{
		{0, "click_interactions_bucket1"},
		{1, "click_interactions_bucket2"},
		{2, "click_interactions_bucket3"},
		{3, ""},
	}

	for _, tt := range tests {
		result := getTableName(tt.bucketIdx)
		assert.Equal(t, tt.expected, result)
	}
}

// Verifies that empty data map returns empty result during deserialization.
func TestClickRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_EmptyData(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(map[string][]byte{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that nil column values are skipped during deserialization.
func TestClickRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_NilData(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": nil,
	}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(weekToData)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that empty byte slices are treated as no data during deserialization.
func TestClickRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_EmptyByteSlice(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": {},
	}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(weekToData)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that invalid/corrupted data returns an error during deserialization.
func TestClickRetrieveHandler_buildDeserialisedPermanentStorageDataBlocks_InvalidData(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": []byte("invalid-data"),
	}

	result, err := handler.buildDeserialisedPermanentStorageDataBlocks(weekToData)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to deserialize")
}

// Verifies that empty table-to-fields mapping returns empty result.
func TestClickRetrieveHandler_fetchDataInParallel_EmptyTableToFields(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{}

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that tables with no fields are skipped and no database call is made.
func TestClickRetrieveHandler_fetchDataInParallel_EmptyFieldsForTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"click_interactions_bucket1": {},
	}

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Empty(t, result)
	mockDb.AssertNotCalled(t, "RetrieveInteractions")
}

// Verifies that data is correctly fetched from a single table with multiple week columns.
func TestClickRetrieveHandler_fetchDataInParallel_SingleTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"click_interactions_bucket1": {"week_0", "week_1"},
	}

	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", []string{"week_0", "week_1"}).
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
func TestClickRetrieveHandler_fetchDataInParallel_DatabaseError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"click_interactions_bucket1": {"week_0"},
	}

	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", []string{"week_0"}).
		Return(nil, errors.New("database error"))

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve data")
	mockDb.AssertExpectations(t)
}

// Verifies that missing columns in database response are handled gracefully as nil.
func TestClickRetrieveHandler_fetchDataInParallel_MissingColumn(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"click_interactions_bucket1": {"week_0", "week_1"},
	}

	// Return data for only week_0, week_1 is missing
	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", []string{"week_0", "week_1"}).
		Return(map[string]interface{}{
			"week_0": []byte{1, 2, 3},
		}, nil)

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	// week_1 should be nil since it's missing
	assert.Nil(t, result["week_1"])
	mockDb.AssertExpectations(t)
}

// Verifies that unexpected data types in database response return an error.
func TestClickRetrieveHandler_fetchDataInParallel_WrongDataType(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"click_interactions_bucket1": {"week_0"},
	}

	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", []string{"week_0"}).
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
func TestClickRetrieveHandler_Retrieve_LimitZero(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, 0)

	assert.ErrorIs(t, err, ErrInvalidLimit)
	assert.Nil(t, events)
	mockDb.AssertExpectations(t)
}

// Verifies that negative limit returns ErrInvalidLimit without calling the DB.
func TestClickRetrieveHandler_Retrieve_NegativeLimit(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, -1)

	assert.ErrorIs(t, err, ErrInvalidLimit)
	assert.Nil(t, events)
	mockDb.AssertExpectations(t)
}

// Verifies that data is fetched in parallel from multiple tables (different buckets).
func TestClickRetrieveHandler_fetchDataInParallel_MultipleTables(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	tablesToFields := map[string][]string{
		"click_interactions_bucket1": {"week_0"},
		"click_interactions_bucket2": {"week_8"},
	}

	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", []string{"week_0"}).
		Return(map[string]interface{}{
			"week_0": []byte{},
		}, nil)

	mockDb.On("RetrieveInteractions", "click_interactions_bucket2", "user1", []string{"week_8"}).
		Return(map[string]interface{}{
			"week_8": []byte{},
		}, nil)

	result, err := handler.fetchDataInParallel(tablesToFields, "user1")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockDb.AssertExpectations(t)
}

// Verifies that single day query only hits one table and one week column.
func TestClickRetrieveHandler_Retrieve_SingleWeekSingleTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	// Single day query - should only query one week
	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.NoError(t, err)
	assert.Empty(t, events)
	// Verify only bucket1 was queried
	mockDb.AssertCalled(t, "RetrieveInteractions", "click_interactions_bucket1", "user1", mock.AnythingOfType("[]string"))
	mockDb.AssertNotCalled(t, "RetrieveInteractions", "click_interactions_bucket2", mock.Anything, mock.Anything)
}

// Verifies that table-to-fields mapping spans multiple buckets for cross-bucket range.
func TestClickRetrieveHandler_buildTableToFieldsMapping_CrossBucketRange(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	startTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 3, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(handler.buildOrderedWeeks(startTime, endTime))

	assert.NotEmpty(t, result)
	assert.GreaterOrEqual(t, len(result), 1)
}

// Verifies that requested limit exceeding max is capped to max limit.
func TestClickRetrieveHandler_Retrieve_CapLimitApplied(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	events, err := handler.Retrieve("user1", startTime, endTime, 5000)

	assert.NoError(t, err)
	assert.Empty(t, events)
	mockDb.AssertExpectations(t)
}

func TestClickRetrieveHandler_Retrieve_TimeRangeExceeded(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	// Use a relative past window (>24 weeks) so we always get ErrTimeRangeExceeded, not ErrEndTimeInFuture
	startTime := time.Now().Add(-26 * 7 * 24 * time.Hour).UnixMilli()
	endTime := time.Now().Add(-1 * time.Hour).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.ErrorIs(t, err, ErrTimeRangeExceeded)
	assert.Nil(t, events)
}

func TestClickRetrieveHandler_Retrieve_EndTimeInFuture(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	startTime := time.Now().Add(-1 * time.Hour).UnixMilli()
	endTime := time.Now().Add(24 * time.Hour).UnixMilli()

	events, err := handler.Retrieve("user1", startTime, endTime, 100)

	assert.ErrorIs(t, err, ErrEndTimeInFuture)
	assert.Nil(t, events)
}

func TestClickRetrieveHandler_buildOrderedWeeks_NormalRange(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	startTime := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC).UnixMilli()

	weeks := handler.buildOrderedWeeks(startTime, endTime)

	assert.NotEmpty(t, weeks)
	// Weeks should be in descending order
	for i := 1; i < len(weeks); i++ {
		assert.Greater(t, weeks[i-1], weeks[i])
	}
}

func TestClickRetrieveHandler_buildOrderedWeeks_SingleWeek(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	weeks := handler.buildOrderedWeeks(startTime, endTime)

	assert.Len(t, weeks, 1)
}

// TestClickRetrieveHandler_buildOrderedWeeks_WrapAround covers the branch when
// lowerBound > upperBound (range crosses week-24 boundary), e.g. week 23 → week 0.
func TestClickRetrieveHandler_buildOrderedWeeks_WrapAround(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	// June 3 2024 = ISO week 23, June 10 = week 24 → 23%24=23, 24%24=0 → wrap branch
	startTime := time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC).UnixMilli()

	weeks := handler.buildOrderedWeeks(startTime, endTime)

	assert.NotEmpty(t, weeks)
	for _, w := range weeks {
		assert.GreaterOrEqual(t, w, 0)
		assert.Less(t, w, 24)
	}
}

func TestClickRetrieveHandler_mergeFilterAndLimit_FiltersAndLimits(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	start := int64(100)
	end := int64(500)

	weekToEvents := map[string][]model.ClickEvent{
		"week_3": {
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 450, CatalogId: 1}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 300, CatalogId: 2}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 200, CatalogId: 3}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 50, CatalogId: 4}}},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, start, end, 10)

	assert.Len(t, result, 3)
	assert.Equal(t, int32(1), result[0].ClickEventData.Payload.CatalogId)
	assert.Equal(t, int32(2), result[1].ClickEventData.Payload.CatalogId)
	assert.Equal(t, int32(3), result[2].ClickEventData.Payload.CatalogId)
}

func TestClickRetrieveHandler_mergeFilterAndLimit_RespectsLimit(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToEvents := map[string][]model.ClickEvent{
		"week_3": {
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 400}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 300}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 200}}},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, 100, 500, 2)

	assert.Len(t, result, 2)
}

func TestClickRetrieveHandler_mergeFilterAndLimit_MultipleWeeks(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToEvents := map[string][]model.ClickEvent{
		"week_4": {
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 900, CatalogId: 1}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 800, CatalogId: 2}}},
		},
		"week_3": {
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 500, CatalogId: 3}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 400, CatalogId: 4}}},
		},
	}

	// Weeks in descending order: 4, 3
	result := handler.mergeFilterAndLimit([]int{4, 3}, weekToEvents, 100, 1000, 10)

	assert.Len(t, result, 4)
	// Should be in descending timestamp order: 900, 800, 500, 400
	assert.Equal(t, int64(900), result[0].ClickEventData.Payload.ClickedAt)
	assert.Equal(t, int64(800), result[1].ClickEventData.Payload.ClickedAt)
	assert.Equal(t, int64(500), result[2].ClickEventData.Payload.ClickedAt)
	assert.Equal(t, int64(400), result[3].ClickEventData.Payload.ClickedAt)
}

func TestClickRetrieveHandler_mergeFilterAndLimit_EmptyWeeks(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToEvents := map[string][]model.ClickEvent{}

	result := handler.mergeFilterAndLimit([]int{3, 4}, weekToEvents, 100, 500, 10)

	assert.Empty(t, result)
}

func TestClickRetrieveHandler_mergeFilterAndLimit_AllEventsOutOfRange(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToEvents := map[string][]model.ClickEvent{
		"week_3": {
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 900}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 800}}},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, 100, 500, 10)

	assert.Empty(t, result)
}

func TestClickRetrieveHandler_mergeFilterAndLimit_SkipsEventsAboveEnd(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToEvents := map[string][]model.ClickEvent{
		"week_3": {
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 600}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 400}}},
			{ClickEventData: model.ClickEventData{Payload: model.ClickEventPayload{ClickedAt: 200}}},
		},
	}

	result := handler.mergeFilterAndLimit([]int{3}, weekToEvents, 100, 500, 10)

	assert.Len(t, result, 2)
	assert.Equal(t, int64(400), result[0].ClickEventData.Payload.ClickedAt)
	assert.Equal(t, int64(200), result[1].ClickEventData.Payload.ClickedAt)
}

func TestClickRetrieveHandler_deserializeWeeks_EmptyData(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": nil,
		"week_1": {},
	}

	result, err := handler.deserializeWeeks(weekToData, "user1")

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestClickRetrieveHandler_deserializeWeeks_InvalidData(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	weekToData := map[string][]byte{
		"week_0": []byte("corrupt"),
	}

	result, err := handler.deserializeWeeks(weekToData, "user1")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to deserialize")
}
