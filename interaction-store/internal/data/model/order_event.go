package model

type OrderPlacedEvent struct {
	OrderKafkaMetaData   OrderKafkaMetaData `json:"meta"`
	OrderPlacedEventData OrderEventData     `json:"data"`
}

type OrderKafkaMetaData struct {
	Timestamp int64 `json:"timestamp"`
}

type OrderEventData struct {
	UserID      string       `json:"user_id"`
	OrderSplits []OrderSplit `json:"order_splits"`
}

type OrderSplit struct {
	OrderDetails []OrderDetail `json:"order_details"`
}

type OrderDetail struct {
	ProductID      int32          `json:"product_id"`
	SubOrderNum    string         `json:"sub_order_num"`
	CatalogDetails CatalogDetails `json:"catalog_details"`
}

type CatalogDetails struct {
	CatalogID int32 `json:"catalog_id"`
}

// FlattenedOrderEvent is the flattened structure used for storage (one per product in an order)
type FlattenedOrderEvent struct {
	CatalogID   int32
	ProductID   int32
	SubOrderNum string
	OrderedAt   int64
	Metadata    string
}

// FlattenOrderPlacedEvent converts a nested OrderPlacedEvent into a slice of FlattenedOrderEvent
// Note: Metadata field is not populated here - it should be populated separately by the consumer
func FlattenOrderPlacedEvent(event OrderPlacedEvent) []FlattenedOrderEvent {
	var flattened []FlattenedOrderEvent
	timestamp := event.OrderKafkaMetaData.Timestamp

	for _, split := range event.OrderPlacedEventData.OrderSplits {
		for _, detail := range split.OrderDetails {
			flattened = append(flattened, FlattenedOrderEvent{
				CatalogID:   int32(detail.CatalogDetails.CatalogID),
				ProductID:   int32(detail.ProductID),
				SubOrderNum: detail.SubOrderNum,
				OrderedAt:   timestamp,
			})
		}
	}
	return flattened
}
