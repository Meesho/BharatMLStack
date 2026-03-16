package persist

import (
	"errors"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/constants"
	blocks "github.com/Meesho/BharatMLStack/interaction-store/internal/data/block"
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

func newTestClickPersistHandler(db *MockDatabase) *ClickPersistHandler {
	return &ClickPersistHandler{
		scyllaDb: db,
	}
}

func createTestClickEvent(userId string, clickedAt int64, catalogId, productId int32) model.ClickEvent {
	return model.ClickEvent{
		KafkaMetaData: model.KafkaMetaData{
			RequestId:        "test-request-id",
			RequestTimestamp: "2024-01-01T00:00:00Z",
		},
		ClickEventData: model.ClickEventData{
			Payload: model.ClickEventPayload{
				UserId:    userId,
				CatalogId: catalogId,
				ProductId: productId,
				ClickedAt: clickedAt,
			},
		},
	}
}

// Verifies successful persistence of click events for a new user with no existing data.
func TestClickPersistHandler_Persist_SuccessNewUser(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickPersistHandler(mockDb)

	// Use a fixed timestamp that corresponds to a specific week
	clickedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.ClickEvent{
		createTestClickEvent("user1", clickedAt, 100, 200),
	}

	// Mock retrieve returns empty (new user)
	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	// Mock update for data (works as upsert in Scylla)
	mockDb.On("UpdateInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	// Mock metadata update (called asynchronously, so we use Maybe())
	mockDb.On("UpdateMetadata", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil).Maybe()

	err := handler.Persist("user1", events)

	assert.NoError(t, err)
	// Only assert on synchronous expectations
	mockDb.AssertCalled(t, "RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string"))
	mockDb.AssertCalled(t, "UpdateInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("map[string]interface {}"))
}

// Verifies that empty event list is handled gracefully without database calls.
func TestClickPersistHandler_Persist_EmptyEvents(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickPersistHandler(mockDb)

	events := []model.ClickEvent{}

	err := handler.Persist("user1", events)

	assert.NoError(t, err)
	mockDb.AssertNotCalled(t, "RetrieveInteractions")
	mockDb.AssertNotCalled(t, "UpdateInteractions")
}

// Verifies that database errors during retrieval are properly propagated.
func TestClickPersistHandler_Persist_RetrieveInteractionsError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickPersistHandler(mockDb)

	clickedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.ClickEvent{
		createTestClickEvent("user1", clickedAt, 100, 200),
	}

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(nil, errors.New("database connection failed"))

	err := handler.Persist("user1", events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve interactions failed")
	mockDb.AssertExpectations(t)
}

// Verifies that database errors during update are properly propagated.
func TestClickPersistHandler_Persist_UpdateInteractionsError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickPersistHandler(mockDb)

	clickedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.ClickEvent{
		createTestClickEvent("user1", clickedAt, 100, 200),
	}

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	mockDb.On("UpdateInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(errors.New("update failed"))

	err := handler.Persist("user1", events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
	mockDb.AssertExpectations(t)
}

// Verifies that events from the same week are partitioned into a single bucket.
func TestClickPersistHandler_partitionEventsByBucket_SingleBucket(t *testing.T) {
	handler := &ClickPersistHandler{}

	clickedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.ClickEvent{
		createTestClickEvent("user1", clickedAt, 100, 200),
		createTestClickEvent("user1", clickedAt+1000, 101, 201),
	}

	bucketEvents, weekFlags := handler.partitionEventsByBucket(events)

	// All events should be in one bucket
	totalEvents := 0
	for _, evts := range bucketEvents {
		totalEvents += len(evts)
	}
	assert.Equal(t, 2, totalEvents)

	// At least one week flag should be set
	hasFlag := false
	for _, flag := range weekFlags {
		if flag {
			hasFlag = true
			break
		}
	}
	assert.True(t, hasFlag)
}

// Verifies that correct week columns are returned for a given bucket based on week flags.
func TestClickPersistHandler_getColumnsForBucket_ReturnsCorrectColumns(t *testing.T) {
	handler := &ClickPersistHandler{}

	weekFlags := make([]bool, 24)
	weekFlags[0] = true
	weekFlags[2] = true
	weekFlags[5] = true

	columns := handler.getColumnsForBucket(0, weekFlags)

	assert.Len(t, columns, 3)
	assert.Contains(t, columns, "week_0")
	assert.Contains(t, columns, "week_2")
	assert.Contains(t, columns, "week_5")
}

// Verifies that no columns are returned when no week flags are set.
func TestClickPersistHandler_getColumnsForBucket_EmptyWhenNoFlags(t *testing.T) {
	handler := &ClickPersistHandler{}

	weekFlags := make([]bool, 24)

	columns := handler.getColumnsForBucket(0, weekFlags)

	assert.Empty(t, columns)
}

// Verifies that bucket 1 correctly maps to weeks 8-15.
func TestClickPersistHandler_getColumnsForBucket_CorrectBucketRange(t *testing.T) {
	handler := &ClickPersistHandler{}

	weekFlags := make([]bool, 24)
	// Set flags in bucket 1 (weeks 8-15)
	weekFlags[8] = true
	weekFlags[10] = true
	weekFlags[15] = true

	columns := handler.getColumnsForBucket(1, weekFlags)

	assert.Len(t, columns, 3)
	assert.Contains(t, columns, "week_8")
	assert.Contains(t, columns, "week_10")
	assert.Contains(t, columns, "week_15")
}

// Verifies that empty data map is handled gracefully during deserialization.
func TestClickPersistHandler_deserializeExistingData_EmptyData(t *testing.T) {
	handler := &ClickPersistHandler{}

	result, err := handler.deserializeExistingData(map[string]interface{}{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that nil column values are skipped during deserialization.
func TestClickPersistHandler_deserializeExistingData_NilValue(t *testing.T) {
	handler := &ClickPersistHandler{}

	data := map[string]interface{}{
		"week_0": nil,
	}

	result, err := handler.deserializeExistingData(data)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that empty byte slices are treated as new user case during deserialization.
func TestClickPersistHandler_deserializeExistingData_EmptyByteSlice(t *testing.T) {
	handler := &ClickPersistHandler{}

	data := map[string]interface{}{
		"week_0": []byte{},
	}

	result, err := handler.deserializeExistingData(data)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that invalid data types in column values return an error.
func TestClickPersistHandler_deserializeExistingData_WrongType(t *testing.T) {
	handler := &ClickPersistHandler{}

	data := map[string]interface{}{
		"week_0": "invalid-type",
	}

	result, err := handler.deserializeExistingData(data)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected type")
}

// Verifies that new events are correctly appended to existing events.
func TestClickPersistHandler_mergeAndTrimEvents_AppendsNewEvent(t *testing.T) {
	handler := &ClickPersistHandler{}

	clickedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.ClickEvent{
		createTestClickEvent("user1", clickedAt, 100, 200),
	}
	newEvent := createTestClickEvent("user1", clickedAt+1000, 101, 201)

	result := handler.mergeAndTrimEvents(existing, newEvent)

	assert.Len(t, result, 2)
}

// Verifies that merged events are sorted by timestamp in descending order (newest first).
func TestClickPersistHandler_mergeAndTrimEvents_SortsByClickedAtDescending(t *testing.T) {
	handler := &ClickPersistHandler{}

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.ClickEvent{
		createTestClickEvent("user1", baseTime, 100, 200),
	}
	newEvent := createTestClickEvent("user1", baseTime+1000, 101, 201)

	result := handler.mergeAndTrimEvents(existing, newEvent)

	assert.Equal(t, baseTime+1000, result[0].ClickEventData.Payload.ClickedAt)
	assert.Equal(t, baseTime, result[1].ClickEventData.Payload.ClickedAt)
}

// Verifies that events are trimmed to max limit per week, keeping newest events.
func TestClickPersistHandler_mergeAndTrimEvents_TrimsToMaxEvents(t *testing.T) {
	handler := &ClickPersistHandler{}

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()

	// Create maxClickEventsPerWeek existing events
	existing := make([]model.ClickEvent, constants.MaxClickEventsPerWeek)
	for i := 0; i < constants.MaxClickEventsPerWeek; i++ {
		existing[i] = createTestClickEvent("user1", baseTime+int64(i*1000), int32(100+i), int32(200+i))
	}

	newEvent := createTestClickEvent("user1", baseTime+int64(constants.MaxClickEventsPerWeek*1000), 999, 999)

	result := handler.mergeAndTrimEvents(existing, newEvent)

	assert.Len(t, result, constants.MaxClickEventsPerWeek)
	// The newest event should be first
	assert.Equal(t, int32(999), result[0].ClickEventData.Payload.ProductId)
}

// Verifies that bucket indices map to correct table names.
func TestGetTableName_ReturnsCorrectTableNames(t *testing.T) {
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

// Verifies that metadata table name is correctly returned.
func TestGetMetadataTableName_ReturnsCorrectName(t *testing.T) {
	result := getMetadataTableName()
	assert.Equal(t, "click_interactions_metadata", result)
}

// Verifies that events are successfully built into permanent storage data block format.
func TestClickPersistHandler_buildPermanentStorageDataBlock_Success(t *testing.T) {
	handler := &ClickPersistHandler{}

	clickedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.ClickEvent{
		createTestClickEvent("user1", clickedAt, 100, 200),
	}

	psdb, err := handler.buildPermanentStorageDataBlock(events)
	assert.NoError(t, err)
	assert.NotNil(t, psdb)

	// Verify serialization works
	data, err := psdb.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Cleanup
	cleanupPSDBs([]*blocks.PermanentStorageDataBlock{psdb})
}

// Verifies that serialization of empty events returns an error.
func TestClickPersistHandler_buildPermanentStorageDataBlock_EmptyEvents(t *testing.T) {
	handler := &ClickPersistHandler{}

	events := []model.ClickEvent{}

	psdb, err := handler.buildPermanentStorageDataBlock(events)
	assert.NoError(t, err) // Build succeeds
	assert.NotNil(t, psdb)

	// Serialization of empty events should return an error
	data, err := psdb.Serialize()
	assert.Error(t, err)
	assert.Nil(t, data)

	// Cleanup
	cleanupPSDBs([]*blocks.PermanentStorageDataBlock{psdb})
}

// Verifies persistence of events from multiple weeks within the same bucket/table.
func TestClickPersistHandler_Persist_MultipleWeeksInSameTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickPersistHandler(mockDb)

	// Create events in different weeks but same bucket (bucket 0: weeks 0-7)
	// Week 1 and Week 3 are both in bucket 0
	week1Time := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 1
	week3Time := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli() // Week 3

	events := []model.ClickEvent{
		createTestClickEvent("user1", week1Time, 100, 200),
		createTestClickEvent("user1", week3Time, 101, 201),
	}

	// Both weeks are in bucket 0, so only one RetrieveInteractions call
	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	// UpdateInteractions should be called with data for multiple weeks
	mockDb.On("UpdateInteractions", "click_interactions_bucket1", "user1", mock.MatchedBy(func(data map[string]interface{}) bool {
		// Should have entries for two different weeks
		return len(data) == 2
	})).Return(nil)

	mockDb.On("UpdateMetadata", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil).Maybe()

	err := handler.Persist("user1", events)

	assert.NoError(t, err)
	mockDb.AssertExpectations(t)
}

// Verifies persistence of events from multiple weeks across different buckets/tables.
func TestClickPersistHandler_Persist_MultipleWeeksInDifferentTables(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestClickPersistHandler(mockDb)

	// Create events in different buckets
	// Week 2 is in bucket 0 (weeks 0-7), Week 10 is in bucket 1 (weeks 8-15)
	week2Time := time.Date(2024, 1, 8, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 2 -> bucket 0
	week10Time := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC).UnixMilli() // Week 10 -> bucket 1

	events := []model.ClickEvent{
		createTestClickEvent("user1", week2Time, 100, 200),
		createTestClickEvent("user1", week10Time, 101, 201),
	}

	// Bucket 0 (click_interactions_bucket1)
	mockDb.On("RetrieveInteractions", "click_interactions_bucket1", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)
	mockDb.On("UpdateInteractions", "click_interactions_bucket1", "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	// Bucket 1 (click_interactions_bucket2)
	mockDb.On("RetrieveInteractions", "click_interactions_bucket2", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)
	mockDb.On("UpdateInteractions", "click_interactions_bucket2", "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	mockDb.On("UpdateMetadata", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil).Maybe()

	err := handler.Persist("user1", events)

	assert.NoError(t, err)
	// Verify both buckets were accessed
	mockDb.AssertCalled(t, "RetrieveInteractions", "click_interactions_bucket1", "user1", mock.AnythingOfType("[]string"))
	mockDb.AssertCalled(t, "RetrieveInteractions", "click_interactions_bucket2", "user1", mock.AnythingOfType("[]string"))
}

// Verifies that existing events are cleared when new event timestamp exceeds 24 weeks.
func TestClickPersistHandler_mergeAndTrimEvents_RolloverAfter24Weeks(t *testing.T) {
	handler := &ClickPersistHandler{}

	// Create existing event with old timestamp
	oldTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.ClickEvent{
		createTestClickEvent("user1", oldTime, 100, 200),
		createTestClickEvent("user1", oldTime+1000, 101, 201),
	}

	// New event is more than 24 weeks later (24 weeks = 168 days)
	// 25 weeks later to ensure rollover
	newTime := oldTime + (25 * 7 * 24 * 60 * 60 * 1000) // 25 weeks in milliseconds
	newEvent := createTestClickEvent("user1", newTime, 999, 999)

	result := handler.mergeAndTrimEvents(existing, newEvent)

	// Existing events should be cleared, only new event remains
	assert.Len(t, result, 1)
	assert.Equal(t, int32(999), result[0].ClickEventData.Payload.ProductId)
	assert.Equal(t, newTime, result[0].ClickEventData.Payload.ClickedAt)
}

// Verifies that existing events are retained when new event is within 24 weeks.
func TestClickPersistHandler_mergeAndTrimEvents_NoRolloverWithin24Weeks(t *testing.T) {
	handler := &ClickPersistHandler{}

	// Create existing event
	oldTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.ClickEvent{
		createTestClickEvent("user1", oldTime, 100, 200),
	}

	// New event is 23 weeks later (less than 24 weeks threshold)
	newTime := oldTime + (23 * 7 * 24 * 60 * 60 * 1000) // 23 weeks in milliseconds
	newEvent := createTestClickEvent("user1", newTime, 999, 999)

	result := handler.mergeAndTrimEvents(existing, newEvent)

	// Both events should be retained (no rollover)
	assert.Len(t, result, 2)
	// Newest event should be first (sorted descending)
	assert.Equal(t, int32(999), result[0].ClickEventData.Payload.ProductId)
	assert.Equal(t, int32(200), result[1].ClickEventData.Payload.ProductId)
}

// Verifies that rollover occurs at exactly 24 weeks boundary (>= 24 triggers rollover).
func TestClickPersistHandler_mergeAndTrimEvents_RolloverExactly24Weeks(t *testing.T) {
	handler := &ClickPersistHandler{}

	oldTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.ClickEvent{
		createTestClickEvent("user1", oldTime, 100, 200),
	}

	// New event is exactly 24 weeks later
	newTime := oldTime + (24 * 7 * 24 * 60 * 60 * 1000) // 24 weeks in milliseconds
	newEvent := createTestClickEvent("user1", newTime, 999, 999)

	result := handler.mergeAndTrimEvents(existing, newEvent)

	// At exactly 24 weeks, rollover should occur (>= 24)
	assert.Len(t, result, 1)
	assert.Equal(t, int32(999), result[0].ClickEventData.Payload.ProductId)
}

// Verifies that events are correctly partitioned across all three storage buckets.
func TestClickPersistHandler_partitionEventsByBucket_AllThreeBuckets(t *testing.T) {
	handler := &ClickPersistHandler{}

	// Create events for each bucket
	// Bucket 0: weeks 0-7, Bucket 1: weeks 8-15, Bucket 2: weeks 16-23
	bucket0Time := time.Date(2024, 1, 8, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 2
	bucket1Time := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 10
	bucket2Time := time.Date(2024, 4, 22, 12, 0, 0, 0, time.UTC).UnixMilli() // Week 17

	events := []model.ClickEvent{
		createTestClickEvent("user1", bucket0Time, 100, 200),
		createTestClickEvent("user1", bucket1Time, 101, 201),
		createTestClickEvent("user1", bucket2Time, 102, 202),
	}

	bucketEvents, weekFlags := handler.partitionEventsByBucket(events)

	// Should have events in 3 different buckets
	assert.Len(t, bucketEvents, 3)

	// Count total week flags set
	flagCount := 0
	for _, flag := range weekFlags {
		if flag {
			flagCount++
		}
	}
	assert.Equal(t, 3, flagCount)
}
