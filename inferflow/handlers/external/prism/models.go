package external

import "time"

const (
	ERROR_TYPE    = "error-type"
	PRISM_API_ERR = "prism-api-error"
	PRISM_MQ_ERR  = "prism-mq-error"
)

type ItemsLoggingData struct {
	IopId             string    `json:"iop_id"`
	UserId            string    `json:"user_id"`
	InferflowConfigId string    `json:"inferflow_config_id"`
	ItemsMeta         ItemsMeta `json:"items_meta"`
	Items             []Item    `json:"items"`
}

type PrismPayload struct {
	IopId             string    `json:"iop_id"`
	UserId            string    `'json:"user_id"`
	InferflowConfigId string    `json:"inferflow_config_id"`
	ItemsMeta         string    `json:"items_meta"`
	Items             string    `json:"items"`
	CreatedAt         time.Time `json:"created_at"`
}

type Item struct {
	Id       string   `json:"id"`
	Features []string `json:"features"`
}

type ItemsMeta struct {
	NUMERIXComputeID string            `json:"numerix_compute_id,omitempty"`
	Headers          map[string]string `json:"headers_map,omitempty"`
	FeatureSchema    []string          `json:"feature_schema"`
}

type PrismConfig struct {
	Host           string `koanf:"prismHost"`
	Port           string `koanf:"prismPort"`
	Username       string `koanf:"username"`
	Password       string `koanf:"password"`
	Timeout        int    `koanf:"timeout"`
	PrismEventMqId int    `koanf:"prismEventMqId"`
}

type PrismEvent struct {
	Event      string         `json:"event"`
	Properties map[string]any `json:"properties"`
}
