package persist

import (
	"errors"
	"testing"
	"time"

	blocks "github.com/Meesho/BharatMLStack/interaction-store/internal/data/block"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestOrderPersistHandler(db *MockDatabase) *OrderPersistHandler {
	return &OrderPersistHandler{
		scyllaDb: db,
	}
}

func createTestFlattenedOrderEvent(orderedAt int64, catalogId, productId int32, subOrderNum string) model.FlattenedOrderEvent {
	return model.FlattenedOrderEvent{
		CatalogID:   catalogId,
		ProductID:   productId,
		SubOrderNum: subOrderNum,
		OrderedAt:   orderedAt,
	}
}

// Verifies successful persistence of order events for a new user with no existing data.
func TestOrderPersistHandler_Persist_SuccessNewUser(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderPersistHandler(mockDb)

	orderedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(orderedAt, 100, 200, "SUB001"),
	}

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

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
func TestOrderPersistHandler_Persist_EmptyEvents(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderPersistHandler(mockDb)

	events := []model.FlattenedOrderEvent{}

	err := handler.Persist("user1", events)

	assert.NoError(t, err)
	mockDb.AssertNotCalled(t, "RetrieveInteractions")
	mockDb.AssertNotCalled(t, "UpdateInteractions")
}

// Verifies that database errors during retrieval are properly propagated.
func TestOrderPersistHandler_Persist_RetrieveInteractionsError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderPersistHandler(mockDb)

	orderedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(orderedAt, 100, 200, "SUB001"),
	}

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(nil, errors.New("database connection failed"))

	err := handler.Persist("user1", events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve interactions failed")
	mockDb.AssertExpectations(t)
}

// Verifies that database errors during update are properly propagated.
func TestOrderPersistHandler_Persist_UpdateInteractionsError(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderPersistHandler(mockDb)

	orderedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(orderedAt, 100, 200, "SUB001"),
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
func TestOrderPersistHandler_partitionEventsByBucket_SingleBucket(t *testing.T) {
	handler := &OrderPersistHandler{}

	orderedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(orderedAt, 100, 200, "SUB001"),
		createTestFlattenedOrderEvent(orderedAt+1000, 101, 201, "SUB002"),
	}

	bucketEvents, weekFlags := handler.partitionEventsByBucket(events)

	totalEvents := 0
	for _, evts := range bucketEvents {
		totalEvents += len(evts)
	}
	assert.Equal(t, 2, totalEvents)

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
func TestOrderPersistHandler_getColumnsForBucket_ReturnsCorrectColumns(t *testing.T) {
	handler := &OrderPersistHandler{}

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
func TestOrderPersistHandler_getColumnsForBucket_EmptyWhenNoFlags(t *testing.T) {
	handler := &OrderPersistHandler{}

	weekFlags := make([]bool, 24)

	columns := handler.getColumnsForBucket(0, weekFlags)

	assert.Empty(t, columns)
}

// Verifies that bucket 1 correctly maps to weeks 8-15.
func TestOrderPersistHandler_getColumnsForBucket_CorrectBucketRange(t *testing.T) {
	handler := &OrderPersistHandler{}

	weekFlags := make([]bool, 24)
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
func TestOrderPersistHandler_deserializeExistingData_EmptyData(t *testing.T) {
	handler := &OrderPersistHandler{}

	result, err := handler.deserializeExistingData(map[string]interface{}{})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that nil column values are skipped during deserialization.
func TestOrderPersistHandler_deserializeExistingData_NilValue(t *testing.T) {
	handler := &OrderPersistHandler{}

	data := map[string]interface{}{
		"week_0": nil,
	}

	result, err := handler.deserializeExistingData(data)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that empty byte slices are treated as new user case during deserialization.
func TestOrderPersistHandler_deserializeExistingData_EmptyByteSlice(t *testing.T) {
	handler := &OrderPersistHandler{}

	data := map[string]interface{}{
		"week_0": []byte{},
	}

	result, err := handler.deserializeExistingData(data)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// Verifies that invalid data types in column values return an error.
func TestOrderPersistHandler_deserializeExistingData_WrongType(t *testing.T) {
	handler := &OrderPersistHandler{}

	data := map[string]interface{}{
		"week_0": "invalid-type",
	}

	result, err := handler.deserializeExistingData(data)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected type")
}

// Verifies that new events are correctly appended to existing events.
func TestOrderPersistHandler_mergeAndTrimEvents_AppendsNewEvent(t *testing.T) {
	handler := &OrderPersistHandler{}

	orderedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(orderedAt, 100, 200, "SUB001"),
	}
	newEvent := createTestFlattenedOrderEvent(orderedAt+1000, 101, 201, "SUB002")

	result := handler.mergeAndTrimEvents(existing, newEvent)

	assert.Len(t, result, 2)
}

// Verifies that merged events are sorted by timestamp in descending order (newest first).
func TestOrderPersistHandler_mergeAndTrimEvents_SortsByOrderedAtDescending(t *testing.T) {
	handler := &OrderPersistHandler{}

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(baseTime, 100, 200, "SUB001"),
	}
	newEvent := createTestFlattenedOrderEvent(baseTime+1000, 101, 201, "SUB002")

	result := handler.mergeAndTrimEvents(existing, newEvent)

	assert.Equal(t, baseTime+1000, result[0].OrderedAt)
	assert.Equal(t, baseTime, result[1].OrderedAt)
}

// Verifies that events are trimmed to max limit per week, keeping newest events.
func TestOrderPersistHandler_mergeAndTrimEvents_TrimsToMaxEvents(t *testing.T) {
	handler := &OrderPersistHandler{}

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()

	existing := make([]model.FlattenedOrderEvent, maxOrderEventsPerWeek)
	for i := 0; i < maxOrderEventsPerWeek; i++ {
		existing[i] = createTestFlattenedOrderEvent(baseTime+int64(i*1000), int32(100+i), int32(200+i), "SUB001")
	}

	newEvent := createTestFlattenedOrderEvent(baseTime+int64(maxOrderEventsPerWeek*1000), 999, 999, "SUB999")

	result := handler.mergeAndTrimEvents(existing, newEvent)

	assert.Len(t, result, maxOrderEventsPerWeek)
	assert.Equal(t, int32(999), result[0].ProductID)
}

// Verifies that bucket indices map to correct table names.
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

// Verifies that metadata table name is correctly returned.
func TestGetOrderMetadataTableName_ReturnsCorrectName(t *testing.T) {
	result := getOrderMetadataTableName()
	assert.Equal(t, "order_interactions_metadata", result)
}

// Verifies that events are successfully built into permanent storage data block format.
func TestOrderPersistHandler_buildPermanentStorageDataBlock_Success(t *testing.T) {
	handler := &OrderPersistHandler{}

	orderedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(orderedAt, 100, 200, "SUB001"),
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
func TestOrderPersistHandler_buildPermanentStorageDataBlock_EmptyEvents(t *testing.T) {
	handler := &OrderPersistHandler{}

	events := []model.FlattenedOrderEvent{}

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

// Verifies persistence of events from different weeks within the same bucket.
func TestOrderPersistHandler_Persist_MultipleEventsInDifferentWeeks(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderPersistHandler(mockDb)

	// Create events in different weeks
	week1Time := time.Date(2024, 1, 8, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 2
	week2Time := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli() // Week 3

	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(week1Time, 100, 200, "SUB001"),
		createTestFlattenedOrderEvent(week2Time, 101, 201, "SUB002"),
	}

	mockDb.On("RetrieveInteractions", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

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

// Verifies persistence of events from multiple weeks within the same bucket/table.
func TestOrderPersistHandler_Persist_MultipleWeeksInSameTable(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderPersistHandler(mockDb)

	// Create events in different weeks but same bucket (bucket 0: weeks 0-7)
	// Week 1 and Week 3 are both in bucket 0
	week1Time := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 1
	week3Time := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli() // Week 3

	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(week1Time, 100, 200, "SUB001"),
		createTestFlattenedOrderEvent(week3Time, 101, 201, "SUB002"),
	}

	// Both weeks are in bucket 0, so only one RetrieveInteractions call
	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)

	// UpdateInteractions should be called with data for multiple weeks
	mockDb.On("UpdateInteractions", "order_interactions_bucket1", "user1", mock.MatchedBy(func(data map[string]interface{}) bool {
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
func TestOrderPersistHandler_Persist_MultipleWeeksInDifferentTables(t *testing.T) {
	mockDb := new(MockDatabase)
	handler := newTestOrderPersistHandler(mockDb)

	// Create events in different buckets
	// Week 2 is in bucket 0 (weeks 0-7), Week 10 is in bucket 1 (weeks 8-15)
	week2Time := time.Date(2024, 1, 8, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 2 -> bucket 0
	week10Time := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC).UnixMilli() // Week 10 -> bucket 1

	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(week2Time, 100, 200, "SUB001"),
		createTestFlattenedOrderEvent(week10Time, 101, 201, "SUB002"),
	}

	// Bucket 0 (order_interactions_bucket1)
	mockDb.On("RetrieveInteractions", "order_interactions_bucket1", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)
	mockDb.On("UpdateInteractions", "order_interactions_bucket1", "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	// Bucket 1 (order_interactions_bucket2)
	mockDb.On("RetrieveInteractions", "order_interactions_bucket2", "user1", mock.AnythingOfType("[]string")).
		Return(map[string]interface{}{}, nil)
	mockDb.On("UpdateInteractions", "order_interactions_bucket2", "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil)

	mockDb.On("UpdateMetadata", mock.AnythingOfType("string"), "user1", mock.AnythingOfType("map[string]interface {}")).
		Return(nil).Maybe()

	err := handler.Persist("user1", events)

	assert.NoError(t, err)
	// Verify both buckets were accessed
	mockDb.AssertCalled(t, "RetrieveInteractions", "order_interactions_bucket1", "user1", mock.AnythingOfType("[]string"))
	mockDb.AssertCalled(t, "RetrieveInteractions", "order_interactions_bucket2", "user1", mock.AnythingOfType("[]string"))
}

// Verifies that existing events are cleared when new event timestamp exceeds 24 weeks.
func TestOrderPersistHandler_mergeAndTrimEvents_RolloverAfter24Weeks(t *testing.T) {
	handler := &OrderPersistHandler{}

	// Create existing event with old timestamp
	oldTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(oldTime, 100, 200, "SUB001"),
		createTestFlattenedOrderEvent(oldTime+1000, 101, 201, "SUB002"),
	}

	// New event is more than 24 weeks later (24 weeks = 168 days)
	// 25 weeks later to ensure rollover
	newTime := oldTime + (25 * 7 * 24 * 60 * 60 * 1000) // 25 weeks in milliseconds
	newEvent := createTestFlattenedOrderEvent(newTime, 999, 999, "SUB999")

	result := handler.mergeAndTrimEvents(existing, newEvent)

	// Existing events should be cleared, only new event remains
	assert.Len(t, result, 1)
	assert.Equal(t, int32(999), result[0].ProductID)
	assert.Equal(t, newTime, result[0].OrderedAt)
}

// Verifies that existing events are retained when new event is within 24 weeks.
func TestOrderPersistHandler_mergeAndTrimEvents_NoRolloverWithin24Weeks(t *testing.T) {
	handler := &OrderPersistHandler{}

	// Create existing event
	oldTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(oldTime, 100, 200, "SUB001"),
	}

	// New event is 23 weeks later (less than 24 weeks threshold)
	newTime := oldTime + (23 * 7 * 24 * 60 * 60 * 1000) // 23 weeks in milliseconds
	newEvent := createTestFlattenedOrderEvent(newTime, 999, 999, "SUB999")

	result := handler.mergeAndTrimEvents(existing, newEvent)

	// Both events should be retained (no rollover)
	assert.Len(t, result, 2)
	// Newest event should be first (sorted descending)
	assert.Equal(t, int32(999), result[0].ProductID)
	assert.Equal(t, int32(200), result[1].ProductID)
}

// Verifies that rollover occurs at exactly 24 weeks boundary (>= 24 triggers rollover).
func TestOrderPersistHandler_mergeAndTrimEvents_RolloverExactly24Weeks(t *testing.T) {
	handler := &OrderPersistHandler{}

	oldTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	existing := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(oldTime, 100, 200, "SUB001"),
	}

	// New event is exactly 24 weeks later
	newTime := oldTime + (24 * 7 * 24 * 60 * 60 * 1000) // 24 weeks in milliseconds
	newEvent := createTestFlattenedOrderEvent(newTime, 999, 999, "SUB999")

	result := handler.mergeAndTrimEvents(existing, newEvent)

	// At exactly 24 weeks, rollover should occur (>= 24)
	assert.Len(t, result, 1)
	assert.Equal(t, int32(999), result[0].ProductID)
}

// Verifies that events are correctly partitioned across all three storage buckets.
func TestOrderPersistHandler_partitionEventsByBucket_AllThreeBuckets(t *testing.T) {
	handler := &OrderPersistHandler{}

	// Create events for each bucket
	// Bucket 0: weeks 0-7, Bucket 1: weeks 8-15, Bucket 2: weeks 16-23
	bucket0Time := time.Date(2024, 1, 8, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 2
	bucket1Time := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC).UnixMilli()  // Week 10
	bucket2Time := time.Date(2024, 4, 22, 12, 0, 0, 0, time.UTC).UnixMilli() // Week 17

	events := []model.FlattenedOrderEvent{
		createTestFlattenedOrderEvent(bucket0Time, 100, 200, "SUB001"),
		createTestFlattenedOrderEvent(bucket1Time, 101, 201, "SUB002"),
		createTestFlattenedOrderEvent(bucket2Time, 102, 202, "SUB003"),
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
