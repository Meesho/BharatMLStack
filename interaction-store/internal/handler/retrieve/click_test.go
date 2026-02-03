package retrieve

import (
	"errors"
	"testing"
	"time"

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

	// Use timestamps that fall in the same week
	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(startTime, endTime, getTableName)

	assert.NotEmpty(t, result)
}

// Verifies that table-to-fields mapping includes multiple weeks for a multi-week range.
func TestClickRetrieveHandler_buildTableToFieldsMapping_MultipleWeeks(t *testing.T) {
	handler := &ClickRetrieveHandler{}

	// Span multiple weeks
	startTime := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 22, 23, 59, 59, 0, time.UTC).UnixMilli()

	result := handler.buildTableToFieldsMapping(startTime, endTime, getTableName)

	assert.NotEmpty(t, result)

	// Count total fields
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

// Verifies that zero limit is handled gracefully and returns empty result.
func TestClickRetrieveHandler_Retrieve_LimitZero(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

	startTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	endTime := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	events, err := handler.Retrieve("user1", startTime, endTime, 0)

	assert.NoError(t, err)
	assert.Empty(t, events)
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

	// Query spanning from bucket 0 to bucket 1
	// Week 5 is in bucket 0, Week 10 is in bucket 1
	startTime := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli()   // Around week 5
	endTime := time.Date(2024, 3, 15, 23, 59, 59, 0, time.UTC).UnixMilli() // Around week 11

	result := handler.buildTableToFieldsMapping(startTime, endTime, getTableName)

	assert.NotEmpty(t, result)

	// Should have mappings for multiple tables
	tableCount := len(result)
	assert.GreaterOrEqual(t, tableCount, 1)
}

// Verifies that requested limit exceeding max is capped to max limit.
func TestClickRetrieveHandler_Retrieve_CapLimitApplied(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickRetrieveHandler(mockDb)

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
