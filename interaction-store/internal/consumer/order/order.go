package order

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/persist"
	"github.com/rs/zerolog/log"
)

var (
	orderConsumer Consumer
	orderOnce     sync.Once
)

type OrderConsumer struct {
	handler persist.PersistHandler
}

func newOrderConsumer() Consumer {
	if orderConsumer == nil {
		orderOnce.Do(func() {
			orderConsumer = &OrderConsumer{
				handler: persist.InitOrderPersistHandler(),
			}
		})
	}
	return orderConsumer
}

func (c *OrderConsumer) Process(events []model.OrderPlacedEvent) error {
	userEvents := make(map[string][]model.FlatOrderEvent)

	for _, event := range events {
		userID := strings.TrimSpace(event.OrderPlacedEventData.UserID)
		if userID == "" {
			log.Error().Msgf("order event missing user_id: %v", event)
			continue
		}

		flattenedEvents := model.FlattenOrderPlacedEvent(event)
		userEvents[userID] = append(userEvents[userID], flattenedEvents...)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(userEvents))

	for id, userScopedEvents := range userEvents {
		wg.Add(1)
		userID := id
		events := userScopedEvents
		go func() {
			defer wg.Done()
			if err := c.handler.Persist(userID, events); err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		var processingError error
		for err := range errChan {
			if processingError == nil {
				processingError = err
			} else {
				processingError = fmt.Errorf("%v; %w", processingError, err)
			}
		}
		log.Error().Err(processingError).Msg("error processing user order events")
		return processingError
	}

	return nil
}
