package consumer

import (
	"encoding/json"

	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/workflow"
	mqConfig "github.com/Meesho/BharatMLStack/skye/pkg/mq/config"
	"github.com/Meesho/BharatMLStack/skye/pkg/mq/consumer"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

func ProcessStatesConsumer(record mqConfig.ConsumerRecord[string, string], c *kafka.Consumer) error {
	log.Info().Msgf("Processing State for %v", record)
	var modelStateExecutorPayload workflow.ModelStateExecutorPayload
	err := json.Unmarshal([]byte(record.Value), &modelStateExecutorPayload)
	if err != nil {
		log.Error().Msgf("Error in Unmarshalling %s", err)
		return err
	}
	stateMachine := workflow.NewStateMachine(workflow.DefaultVersion)
	err = stateMachine.ProcessStates(&modelStateExecutorPayload)
	if err != nil {
		log.Error().Msgf("Error in Processing State %s", err)
		return err
	}
	err = consumer.Commit(c)
	if err != nil {
		return err
	}
	return nil
}
