package consumer

import (
	"encoding/json"

	"github.com/Meesho/BharatMLStack/skye/internal/admin/handler/workflow"
	skafka "github.com/Meesho/BharatMLStack/skye/pkg/kafka"
	"github.com/rs/zerolog/log"
)

// ProcessStatesConsumer processes a single state record.
func ProcessStatesConsumer(record skafka.ConsumerRecord[string, string]) error {
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
	return nil
}
