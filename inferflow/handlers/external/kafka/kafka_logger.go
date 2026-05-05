package kafka

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
	kafkaWriter *kafka.Writer
)

func getMetricTags(metricTags []string, errType string) []string {
	metricTags = append(metricTags, "error-type:"+errType)
	return metricTags
}

// InitKafkaLogger initializes Kafka writers for inference logging.
func InitKafkaLogger(appConfigs *configs.AppConfigs) {
	bootstrapServers := appConfigs.Configs.KafkaBootstrapServers
	if bootstrapServers == "" {
		logger.Info("Kafka bootstrap servers not configured, inference logging disabled")
		return
	}

	if topic := appConfigs.Configs.KafkaLoggingTopic; topic != "" {
		kafkaWriter = &kafka.Writer{
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
	if kafkaWriter == nil {
		return
	}
	if msg == nil {
		logger.Error("Empty proto message for V2 log", fmt.Errorf("model_id: %s", modelId))
		return
	}

	data, err := proto.Marshal(msg)
	metricTags := []string{"model-id", modelId}
	if err != nil {
		logger.Error("Error marshalling proto for V2 log:", err)
		metrics.Count("inferflow.logging.error", 1, getMetricTags(metricTags, PROTO_MARSHAL_ERR))
		return
	}

	if err := kafkaWriter.WriteMessages(context.Background(), kafka.Message{Value: data}); err != nil {
		logger.Error("Error sending V2 log to Kafka:", err)
		metrics.Count("inferflow.logging.error", 1, getMetricTags(metricTags, KAFKA_V2_ERR))
		return
	}

	metrics.Count("inferflow.logging.kafka_sent", 1, metricTags)
}

// PublishInferenceInsightsLog is kept as an exported alias for cross-package callers.
var PublishInferenceInsightsLog = publishInferenceInsightsLog

func CloseKafkaLogger() {
	if kafkaWriter != nil {
		if err := kafkaWriter.Close(); err != nil {
			logger.Error("Error closing Kafka V2 writer:", err)
		}
	}
}
