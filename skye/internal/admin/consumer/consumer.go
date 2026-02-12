package consumer

import (
	"encoding/json"

	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/workflow"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
)

func ProcessStatesConsumer(record []kafka.Message, c *kafka.Consumer) error {
	log.Info().Msgf("Processing State for %v", record)
	var modelStateExecutorPayload workflow.ModelStateExecutorPayload
	err := json.Unmarshal([]byte(record[0].Value), &modelStateExecutorPayload)
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
	_, err = c.CommitMessage(&record[len(record)-1])
	if err != nil {
		return err
	}
	return nil
}
