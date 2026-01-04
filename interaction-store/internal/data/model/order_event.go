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
