package external

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	kafka "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	eventName = "inferflow_inference_logs"

	iopIdKey   = "iop_id"
	userIdKey  = "user_id"
	modelIdKey = "model_proxy_config_id"
	itemsKey   = "items"
	itemsMeta  = "items_meta"
	createdAt  = "created_at"
)

var (
	v1Writer *kafka.Writer
	v2Writer *kafka.Writer

	v1MetricTags = []string{"transport:kafka", "version:v1"}
	v2MetricTags = []string{"transport:kafka", "version:v2"}
)

// InitKafkaLogger initializes Kafka writers for inference logging.
func InitKafkaLogger(appConfigs *configs.AppConfigs) {
	bootstrapServers := appConfigs.Configs.KafkaBootstrapServers
	if bootstrapServers == "" {
		logger.Info("Kafka bootstrap servers not configured, inference logging disabled")
		return
	}

	if topic := appConfigs.Configs.KafkaV1LogTopic; topic != "" {
		v1Writer = &kafka.Writer{
			Addr:         kafka.TCP(bootstrapServers),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			Async:        true,
		}
		logger.Info(fmt.Sprintf("Kafka V1 writer initialised for topic: %s", topic))
	}

	if topic := appConfigs.Configs.KafkaV2LogTopic; topic != "" {
		v2Writer = &kafka.Writer{
			Addr:         kafka.TCP(bootstrapServers),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			Async:        true,
		}
		logger.Info(fmt.Sprintf("Kafka V2 writer initialised for topic: %s", topic))
	}

	logger.Info("Kafka inference logger initialised")
}

// SendV1Log sends V1 logging data (JSON) to Kafka in batches.
func SendV1Log(data *ItemsLoggingData, batchSize int) {
	if v1Writer == nil {
		return
	}
	if batchSize <= 0 {
		batchSize = 500
	}

	total := len(data.Items)
	if total == 0 {
		return
	}

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		chunk := &ItemsLoggingData{
			IopId:              data.IopId,
			UserId:             data.UserId,
			ModelProxyConfigId: data.ModelProxyConfigId,
			ItemsMeta:          data.ItemsMeta,
			Items:              data.Items[i:end],
		}

		payload, err := buildV1Payload(chunk)
		if err != nil {
			logger.Error("Error building V1 payload:", err)
			continue
		}

		if err := v1Writer.WriteMessages(context.Background(), kafka.Message{Value: payload}); err != nil {
			logger.Error("Error sending V1 log to Kafka:", err)
			metrics.Count("inferflow.logging.error", 1, append(v1MetricTags, ERROR_TYPE, KAFKA_SEND_ERR))
		}
	}
}

// SendV2Log sends V2 logging data (protobuf) to Kafka.
func SendV2Log(msg proto.Message, modelId string) {
	if v2Writer == nil {
		return
	}
	if msg == nil {
		logger.Error("Empty proto message for V2 log", fmt.Errorf("model_id: %s", modelId))
		return
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		logger.Error("Error marshalling proto for V2 log:", err)
		metrics.Count("inferflow.logging.error", 1, append(v2MetricTags, ERROR_TYPE, PROTO_MARSHAL_ERR))
		return
	}

	if err := v2Writer.WriteMessages(context.Background(), kafka.Message{Value: data}); err != nil {
		logger.Error("Error sending V2 log to Kafka:", err)
		metrics.Count("inferflow.logging.error", 1, append(v2MetricTags, ERROR_TYPE, KAFKA_V2_ERR))
		return
	}

	metrics.Count("inferflow.logging.kafka_sent", 1, []string{"model-id", modelId})
}

// buildV1Payload marshals a LogEvent wrapping the items data.
func buildV1Payload(data *ItemsLoggingData) ([]byte, error) {
	itemsJSON, err := json.Marshal(data.Items)
	if err != nil {
		return nil, err
	}
	metaJSON, err := json.Marshal(data.ItemsMeta)
	if err != nil {
		return nil, err
	}

	event := LogEvent{
		Event: eventName,
		Properties: map[string]any{
			iopIdKey:   data.IopId,
			userIdKey:  data.UserId,
			modelIdKey: data.ModelProxyConfigId,
			itemsKey:   string(itemsJSON),
			itemsMeta:  string(metaJSON),
			createdAt:  time.Now().UTC(),
		},
		CreatedAt: time.Now().UTC(),
	}

	return json.Marshal(event)
}
