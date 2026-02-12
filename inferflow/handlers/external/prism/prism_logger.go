package external

import (
	"context"
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	kafka "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

var (
	v2Writer *kafka.Writer

	v2MetricTags = []string{"transport:kafka", "version:v2"}
)

// InitKafkaLogger initializes Kafka writers for inference logging.
func InitKafkaLogger(appConfigs *configs.AppConfigs) {
	bootstrapServers := appConfigs.Configs.KafkaBootstrapServers
	if bootstrapServers == "" {
		logger.Info("Kafka bootstrap servers not configured, inference logging disabled")
		return
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

// publishInferenceInsightsLog sends inference insights (protobuf) to Kafka.
func publishInferenceInsightsLog(msg proto.Message, modelId string) {
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

// PublishInferenceInsightsLog is kept as an exported alias for cross-package callers.
var PublishInferenceInsightsLog = publishInferenceInsightsLog
