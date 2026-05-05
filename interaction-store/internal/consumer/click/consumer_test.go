package click

import (
	"errors"
	"testing"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/persist"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	m.Run()
}

// MockPersistHandler is a mock implementation of PersistHandler for testing
type MockPersistHandler struct {
	mock.Mock
}

func (m *MockPersistHandler) Persist(userId string, data []model.ClickEvent) error {
	args := m.Called(userId, data)
	return args.Error(0)
}

// newTestClickConsumer creates a ClickConsumer with injected mock handler for testing
func newTestClickConsumer(handler persist.PersistHandler[[]model.ClickEvent]) *ClickConsumer {
	return &ClickConsumer{
		handler: handler,
	}
}

func createClickEvent(userId, anonymousUserId string, clickedAt int64, catalogId, productId int32) model.ClickEvent {
	return model.ClickEvent{
		KafkaMetaData: model.KafkaMetaData{
			RequestId:        "test-request-id",
			RequestTimestamp: "2024-01-01T00:00:00Z",
		},
		ClickEventData: model.ClickEventData{
			Payload: model.ClickEventPayload{
				UserId:          userId,
				AnonymousUserId: anonymousUserId,
				CatalogId:       catalogId,
				ProductId:       productId,
				ClickedAt:       clickedAt,
			},
		},
	}
}

func TestClickConsumer_Process_SuccessWithSingleUser(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{
		createClickEvent("user1", "", 1704067200000, 100, 200),
		createClickEvent("user1", "", 1704067201000, 101, 201),
	}

	mockHandler.On("Persist", "user1", mock.AnythingOfType("[]model.ClickEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestClickConsumer_Process_SuccessWithMultipleUsers(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{
		createClickEvent("user1", "", 1704067200000, 100, 200),
		createClickEvent("user2", "", 1704067201000, 101, 201),
		createClickEvent("user1", "", 1704067202000, 102, 202),
	}

	mockHandler.On("Persist", "user1", mock.AnythingOfType("[]model.ClickEvent")).Return(nil)
	mockHandler.On("Persist", "user2", mock.AnythingOfType("[]model.ClickEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestClickConsumer_Process_EmptyEvents(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{}

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertNotCalled(t, "Persist")
}

func TestClickConsumer_Process_UsesAnonymousUserIdWhenUserIdMissing(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{
		createClickEvent("", "anon-user-1", 1704067200000, 100, 200),
	}

	mockHandler.On("Persist", "anon-user-1", mock.AnythingOfType("[]model.ClickEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestClickConsumer_Process_SkipsEventsWithoutAnyUserId(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{
		createClickEvent("", "", 1704067200000, 100, 200),
		createClickEvent("user1", "", 1704067201000, 101, 201),
	}

	mockHandler.On("Persist", "user1", mock.AnythingOfType("[]model.ClickEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestClickConsumer_Process_ReturnsErrorOnPersistFailure(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{
		createClickEvent("user1", "", 1704067200000, 100, 200),
	}

	expectedErr := errors.New("database connection failed")
	mockHandler.On("Persist", "user1", mock.AnythingOfType("[]model.ClickEvent")).Return(expectedErr)

	err := consumer.Process(events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persist failed for user user1")
	mockHandler.AssertExpectations(t)
}

func TestClickConsumer_Process_PartialFailure(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{
		createClickEvent("user1", "", 1704067200000, 100, 200),
		createClickEvent("user2", "", 1704067201000, 101, 201),
	}

	mockHandler.On("Persist", "user1", mock.AnythingOfType("[]model.ClickEvent")).Return(nil)
	mockHandler.On("Persist", "user2", mock.AnythingOfType("[]model.ClickEvent")).Return(errors.New("persist error"))

	err := consumer.Process(events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persist failed for user user2")
	mockHandler.AssertExpectations(t)
}

func TestClickConsumer_Process_TrimsWhitespaceFromUserId(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestClickConsumer(mockHandler)

	events := []model.ClickEvent{
		createClickEvent("  user1  ", "", 1704067200000, 100, 200),
	}

	mockHandler.On("Persist", "user1", mock.AnythingOfType("[]model.ClickEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestClickConsumer_extractUserID_ReturnsUserIdWhenPresent(t *testing.T) {
	consumer := &ClickConsumer{}
	event := createClickEvent("user1", "anon-user", 1704067200000, 100, 200)

	userID := consumer.extractUserID(event)

	assert.Equal(t, "user1", userID)
}

func TestClickConsumer_extractUserID_ReturnsAnonymousUserIdWhenUserIdEmpty(t *testing.T) {
	consumer := &ClickConsumer{}
	event := createClickEvent("", "anon-user", 1704067200000, 100, 200)

	userID := consumer.extractUserID(event)

	assert.Equal(t, "anon-user", userID)
}

func TestClickConsumer_extractUserID_ReturnsEmptyWhenBothMissing(t *testing.T) {
	consumer := &ClickConsumer{}
	event := createClickEvent("", "", 1704067200000, 100, 200)

	userID := consumer.extractUserID(event)

	assert.Empty(t, userID)
}

func TestClickConsumer_extractUserID_TrimsWhitespace(t *testing.T) {
	consumer := &ClickConsumer{}
	event := createClickEvent("   ", "  anon-user  ", 1704067200000, 100, 200)

	userID := consumer.extractUserID(event)

	assert.Equal(t, "anon-user", userID)
}

func TestClickConsumer_preprocessAndValidateEvents_GroupsByUserId(t *testing.T) {
	consumer := &ClickConsumer{}
	events := []model.ClickEvent{
		createClickEvent("user1", "", 1704067200000, 100, 200),
		createClickEvent("user2", "", 1704067201000, 101, 201),
		createClickEvent("user1", "", 1704067202000, 102, 202),
	}

	userEvents := consumer.preprocessAndValidateEvents(events)

	assert.Len(t, userEvents, 2)
	assert.Len(t, userEvents["user1"], 2)
	assert.Len(t, userEvents["user2"], 1)
}

func TestClickConsumer_preprocessAndValidateEvents_FiltersInvalidEvents(t *testing.T) {
	consumer := &ClickConsumer{}
	events := []model.ClickEvent{
		createClickEvent("user1", "", 1704067200000, 100, 200),
		createClickEvent("", "", 1704067201000, 101, 201),
		createClickEvent("", "anon-user", 1704067202000, 102, 202),
	}

	userEvents := consumer.preprocessAndValidateEvents(events)

	assert.Len(t, userEvents, 2)
	assert.Contains(t, userEvents, "user1")
	assert.Contains(t, userEvents, "anon-user")
}

func TestClickConsumer_persistUserEvents_ReturnsNilForEmptyMap(t *testing.T) {
	consumer := &ClickConsumer{}

	err := consumer.persistUserEvents(map[string][]model.ClickEvent{})

	assert.NoError(t, err)
}
