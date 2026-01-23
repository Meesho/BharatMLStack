package model

type ClickEvent struct {
	KafkaMetaData  KafkaMetaData  `json:"meta"`
	ClickEventData ClickEventData `json:"data"`
}

type KafkaMetaData struct {
	RequestId        string `json:"requestId"`
	RequestTimestamp string `json:"requestTimestamp"`
}

type ClickEventData struct {
	Payload ClickEventPayload `json:"payload"`
}

type ClickEventPayload struct {
	UserId          string `json:"user_id"`
	AnonymousUserId string `json:"anonymous_user_id"`
	CatalogId       int32  `json:"catalog_id"`
	ProductId       int32  `json:"product_id"`
	ClickedAt       int64  `json:"clicked_at"`
	Metadata        string
}
