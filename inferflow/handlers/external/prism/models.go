package external

import "time"

const (
	ERROR_TYPE        = "error-type"
	KAFKA_SEND_ERR    = "kafka-send-error"
	KAFKA_V2_ERR      = "kafka-v2-error"
	PROTO_MARSHAL_ERR = "proto-marshal-error"
)

// ItemsLoggingData holds the V1 JSON logging payload.
type ItemsLoggingData struct {
	IopId              string    `json:"iop_id"`
	UserId             string    `json:"user_id"`
	ModelProxyConfigId string    `json:"model_proxy_config_id"`
	ItemsMeta          ItemsMeta `json:"items_meta"`
	Items              []Item    `json:"items"`
}

type Item struct {
	Id       string   `json:"id"`
	Features []string `json:"features"`
}

type ItemsMeta struct {
	ComputeID     string            `json:"compute_id,omitempty"`
	Headers       map[string]string `json:"headers_map,omitempty"`
	FeatureSchema []string          `json:"feature_schema"`
}

// LogEvent is the envelope sent to Kafka for V1 logging.
type LogEvent struct {
	Event      string         `json:"event"`
	Properties map[string]any `json:"properties"`
	CreatedAt  time.Time      `json:"created_at"`
}
