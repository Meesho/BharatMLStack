package order

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

func (m *MockPersistHandler) Persist(userId string, data []model.FlattenedOrderEvent) error {
	args := m.Called(userId, data)
	return args.Error(0)
}

// newTestOrderConsumer creates an OrderConsumer with injected mock handler for testing
func newTestOrderConsumer(handler persist.PersistHandler[[]model.FlattenedOrderEvent]) *OrderConsumer {
	return &OrderConsumer{
		handler: handler,
	}
}

func createOrderPlacedEvent(userId int64, timestamp int64, products []struct {
	catalogId   int32
	productId   int32
	subOrderNum string
}) model.OrderPlacedEvent {
	orderDetails := make([]model.OrderDetail, len(products))
	for i, p := range products {
		orderDetails[i] = model.OrderDetail{
			ProductID:   p.productId,
			SubOrderNum: p.subOrderNum,
			CatalogDetails: model.CatalogDetails{
				CatalogID: p.catalogId,
			},
		}
	}

	return model.OrderPlacedEvent{
		OrderKafkaMetaData: model.OrderKafkaMetaData{
			Timestamp: timestamp,
		},
		OrderPlacedEventData: model.OrderEventData{
			UserID: userId,
			OrderSplits: []model.OrderSplit{
				{
					OrderDetails: orderDetails,
				},
			},
		},
	}
}

func createSimpleOrderPlacedEvent(userId int64, timestamp int64, catalogId, productId int32) model.OrderPlacedEvent {
	return createOrderPlacedEvent(userId, timestamp, []struct {
		catalogId   int32
		productId   int32
		subOrderNum string
	}{
		{catalogId: catalogId, productId: productId, subOrderNum: "SUB001"},
	})
}

func TestOrderConsumer_Process_SuccessWithSingleUser(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(1001, 1704067200000, 100, 200),
		createSimpleOrderPlacedEvent(1001, 1704067201000, 101, 201),
	}

	mockHandler.On("Persist", "1001", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestOrderConsumer_Process_SuccessWithMultipleUsers(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(1001, 1704067200000, 100, 200),
		createSimpleOrderPlacedEvent(1002, 1704067201000, 101, 201),
		createSimpleOrderPlacedEvent(1001, 1704067202000, 102, 202),
	}

	mockHandler.On("Persist", "1001", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(nil)
	mockHandler.On("Persist", "1002", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestOrderConsumer_Process_EmptyEvents(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{}

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertNotCalled(t, "Persist")
}

func TestOrderConsumer_Process_SkipsEventsWithoutUserId(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(0, 1704067200000, 100, 200),
		createSimpleOrderPlacedEvent(1001, 1704067201000, 101, 201),
	}

	mockHandler.On("Persist", "1001", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestOrderConsumer_Process_ReturnsErrorOnPersistFailure(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(1001, 1704067200000, 100, 200),
	}

	expectedErr := errors.New("database connection failed")
	mockHandler.On("Persist", "1001", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(expectedErr)

	err := consumer.Process(events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persist failed for user 1001")
	mockHandler.AssertExpectations(t)
}

func TestOrderConsumer_Process_PartialFailure(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(1001, 1704067200000, 100, 200),
		createSimpleOrderPlacedEvent(1002, 1704067201000, 101, 201),
	}

	mockHandler.On("Persist", "1001", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(nil)
	mockHandler.On("Persist", "1002", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(errors.New("persist error"))

	err := consumer.Process(events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persist failed for user 1002")
	mockHandler.AssertExpectations(t)
}

func TestOrderConsumer_Process_HandlesLargeUserId(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(9223372036854775807, 1704067200000, 100, 200),
	}

	mockHandler.On("Persist", "9223372036854775807", mock.AnythingOfType("[]model.FlattenedOrderEvent")).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestOrderConsumer_Process_FlattensMultipleProductsInOrder(t *testing.T) {
	mockHandler := new(MockPersistHandler)
	consumer := newTestOrderConsumer(mockHandler)

	events := []model.OrderPlacedEvent{
		createOrderPlacedEvent(1001, 1704067200000, []struct {
			catalogId   int32
			productId   int32
			subOrderNum string
		}{
			{catalogId: 100, productId: 200, subOrderNum: "SUB001"},
			{catalogId: 101, productId: 201, subOrderNum: "SUB002"},
			{catalogId: 102, productId: 202, subOrderNum: "SUB003"},
		}),
	}

	mockHandler.On("Persist", "1001", mock.MatchedBy(func(data []model.FlattenedOrderEvent) bool {
		return len(data) == 3
	})).Return(nil)

	err := consumer.Process(events)

	assert.NoError(t, err)
	mockHandler.AssertExpectations(t)
}

func TestOrderConsumer_extractAndValidateUserID_ReturnsUserIdWhenPresent(t *testing.T) {
	consumer := &OrderConsumer{}
	event := createSimpleOrderPlacedEvent(1001, 1704067200000, 100, 200)

	userID := consumer.extractAndValidateUserID(event)

	assert.Equal(t, "1001", userID)
}

func TestOrderConsumer_extractAndValidateUserID_ReturnsEmptyWhenZero(t *testing.T) {
	consumer := &OrderConsumer{}
	event := createSimpleOrderPlacedEvent(0, 1704067200000, 100, 200)

	userID := consumer.extractAndValidateUserID(event)

	assert.Empty(t, userID)
}

func TestOrderConsumer_extractAndValidateUserID_HandlesLargeIds(t *testing.T) {
	consumer := &OrderConsumer{}
	event := createSimpleOrderPlacedEvent(9223372036854775807, 1704067200000, 100, 200)

	userID := consumer.extractAndValidateUserID(event)

	assert.Equal(t, "9223372036854775807", userID)
}

func TestOrderConsumer_preprocessAndValidateEvents_GroupsByUserId(t *testing.T) {
	consumer := &OrderConsumer{}
	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(1001, 1704067200000, 100, 200),
		createSimpleOrderPlacedEvent(1002, 1704067201000, 101, 201),
		createSimpleOrderPlacedEvent(1001, 1704067202000, 102, 202),
	}

	userEvents := consumer.preprocessAndValidateEvents(events)

	assert.Len(t, userEvents, 2)
	assert.Len(t, userEvents["1001"], 2)
	assert.Len(t, userEvents["1002"], 1)
}

func TestOrderConsumer_preprocessAndValidateEvents_FiltersInvalidEvents(t *testing.T) {
	consumer := &OrderConsumer{}
	events := []model.OrderPlacedEvent{
		createSimpleOrderPlacedEvent(1001, 1704067200000, 100, 200),
		createSimpleOrderPlacedEvent(0, 1704067201000, 101, 201),
	}

	userEvents := consumer.preprocessAndValidateEvents(events)

	assert.Len(t, userEvents, 1)
	assert.Contains(t, userEvents, "1001")
}

func TestOrderConsumer_persistUserEvents_ReturnsNilForEmptyMap(t *testing.T) {
	consumer := &OrderConsumer{}

	err := consumer.persistUserEvents(map[string][]model.FlattenedOrderEvent{})

	assert.NoError(t, err)
}

func TestOrderConsumer_preprocessAndValidateEvents_FlattensOrderEvents(t *testing.T) {
	consumer := &OrderConsumer{}
	events := []model.OrderPlacedEvent{
		createOrderPlacedEvent(1001, 1704067200000, []struct {
			catalogId   int32
			productId   int32
			subOrderNum string
		}{
			{catalogId: 100, productId: 200, subOrderNum: "SUB001"},
			{catalogId: 101, productId: 201, subOrderNum: "SUB002"},
		}),
	}

	userEvents := consumer.preprocessAndValidateEvents(events)

	assert.Len(t, userEvents["1001"], 2)
	assert.Equal(t, int32(100), userEvents["1001"][0].CatalogID)
	assert.Equal(t, int32(101), userEvents["1001"][1].CatalogID)
}
