package model

type OrderPlacedEvent struct {
	OrderKafkaMetaData   OrderKafkaMetaData `json:"meta"`
	OrderPlacedEventData OrderEventData     `json:"data"`
}

type OrderKafkaMetaData struct {
	Timestamp int64  `json:"timestamp"` // In milliseconds
	RequestID string `json:"request_id"`
	Service   string `json:"service"`
}

type OrderEventData struct {
	UserID      string       `json:"user_id"`
	OrderSplits []OrderSplit `json:"order_splits"`
}

type OrderSplit struct {
	OrderDetails []OrderDetail `json:"order_details"`
}

type OrderDetail struct {
	ProductID      int            `json:"product_id"`
	SubOrderNum    string         `json:"sub_order_num"`
	CatalogDetails CatalogDetails `json:"catalog_details"`
}

type CatalogDetails struct {
	CatalogID int `json:"catalog_id"`
}

// FlatOrderEvent is the flattened structure used for storage (one per product in an order)
type FlatOrderEvent struct {
	CatalogID   int32
	ProductID   int32
	SubOrderNum string
	OrderedAt   int64
}

// FlattenOrderPlacedEvent converts a nested OrderPlacedEvent into a slice of FlatOrderEvent
func FlattenOrderPlacedEvent(event OrderPlacedEvent) []FlatOrderEvent {
	var flattened []FlatOrderEvent
	timestamp := event.OrderKafkaMetaData.Timestamp

	for _, split := range event.OrderPlacedEventData.OrderSplits {
		for _, detail := range split.OrderDetails {
			flattened = append(flattened, FlatOrderEvent{
				CatalogID:   int32(detail.CatalogDetails.CatalogID),
				ProductID:   int32(detail.ProductID),
				SubOrderNum: detail.SubOrderNum,
				OrderedAt:   timestamp,
			})
		}
	}
	return flattened
}
