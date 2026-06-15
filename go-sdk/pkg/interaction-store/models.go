package interactionstore

import (
	pb "github.com/Meesho/BharatMLStack/go-sdk/pkg/proto/interaction-store/timeseries"
)

type Config struct {
	Host      string
	Port      string
	DeadLine  int
	PlainText bool
	CallerId  string

	// Optional client-side gRPC keepalive. Keepalive is opt-in: when KeepaliveTimeMs
	// is <= 0 (the zero value) it stays disabled and the connection behaves exactly as
	// before. When set, the client sends an HTTP/2 keepalive PING every KeepaliveTimeMs
	// and waits KeepaliveTimeoutMs for the ack before treating the connection as dead.
	// KeepalivePermitWithoutStream allows pings on a connection that has no active RPCs.
	// When keepalive is enabled, KeepaliveTimeoutMs must be > 0; a zero or negative
	// timeout is rejected at client construction (it would otherwise be passed to gRPC
	// as a non-positive timer and fail keepalive immediately).
	//
	// Callers MUST keep KeepaliveTimeMs at or above the server's keepalive
	// EnforcementPolicy MinTime; pinging more frequently than the server permits causes
	// the server to close the connection with a GOAWAY "too_many_pings".
	KeepaliveTimeMs              int
	KeepaliveTimeoutMs           int
	KeepalivePermitWithoutStream bool
}

type InteractionType int

const (
	InteractionTypeClick InteractionType = 0
	InteractionTypeOrder InteractionType = 1
)

type PersistClickDataRequest struct {
	UserId string      `json:"user_id"`
	Data   []ClickData `json:"data"`
}

type ClickData struct {
	CatalogId int32  `json:"catalog_id"`
	ProductId int32  `json:"product_id"`
	Timestamp int64  `json:"timestamp"`
	Metadata  string `json:"metadata"`
}

type PersistOrderDataRequest struct {
	UserId string      `json:"user_id"`
	Data   []OrderData `json:"data"`
}

type OrderData struct {
	CatalogId   int32  `json:"catalog_id"`
	ProductId   int32  `json:"product_id"`
	SubOrderNum string `json:"sub_order_num"`
	Timestamp   int64  `json:"timestamp"`
	Metadata    string `json:"metadata"`
}

type PersistDataResponse struct {
	Message string `json:"message"`
}

type RetrieveDataRequest struct {
	UserId         string `json:"user_id"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
	Limit          int32  `json:"limit"`
}

type RetrieveInteractionsRequest struct {
	UserId           string            `json:"user_id"`
	InteractionTypes []InteractionType `json:"interaction_types"`
	StartTimestamp   int64             `json:"start_timestamp"`
	EndTimestamp     int64             `json:"end_timestamp"`
	Limit            int32             `json:"limit"`
}

type ClickEvent struct {
	CatalogId int32  `json:"catalog_id"`
	ProductId int32  `json:"product_id"`
	Timestamp int64  `json:"timestamp"`
	Metadata  string `json:"metadata"`
}

type OrderEvent struct {
	CatalogId   int32  `json:"catalog_id"`
	ProductId   int32  `json:"product_id"`
	SubOrderNum string `json:"sub_order_num"`
	Timestamp   int64  `json:"timestamp"`
	Metadata    string `json:"metadata"`
}

type RetrieveClickDataResponse struct {
	Data []ClickEvent `json:"data"`
}

type RetrieveOrderDataResponse struct {
	Data []OrderEvent `json:"data"`
}

type InteractionData struct {
	ClickEvents []ClickEvent `json:"click_events"`
	OrderEvents []OrderEvent `json:"order_events"`
}

type RetrieveInteractionsResponse struct {
	Data map[string]InteractionData `json:"data"`
}

type Response struct {
	Err  string `json:"err"`
	Resp any    `json:"resp"`
}

type ClickResponse struct {
	Err  string                        `json:"err"`
	Resp *pb.RetrieveClickDataResponse `json:"resp"`
}

type OrderResponse struct {
	Err  string                        `json:"err"`
	Resp *pb.RetrieveOrderDataResponse `json:"resp"`
}

type InteractionsResponse struct {
	Err  string                           `json:"err"`
	Resp *pb.RetrieveInteractionsResponse `json:"resp"`
}
