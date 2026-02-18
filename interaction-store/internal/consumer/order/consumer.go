package order

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/persist"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Consumer interface {
	Process(events []model.OrderPlacedEvent) error
}

var (
	orderConsumer Consumer
	syncOnce      sync.Once
)

type OrderConsumer struct {
	handler persist.PersistHandler[[]model.FlattenedOrderEvent]
}

func newOrderConsumer() Consumer {
	if orderConsumer == nil {
		syncOnce.Do(func() {
			orderConsumer = &OrderConsumer{
				handler: persist.InitOrderPersistHandler(),
			}
		})
	}
	return orderConsumer
}

func (c *OrderConsumer) Process(events []model.OrderPlacedEvent) error {
	userEvents := c.preprocessAndValidateEvents(events)
	return c.persistUserEvents(userEvents)
}

// preprocessAndValidateEvents validates and groups order events by user ID
func (c *OrderConsumer) preprocessAndValidateEvents(events []model.OrderPlacedEvent) map[string][]model.FlattenedOrderEvent {
	userEvents := make(map[string][]model.FlattenedOrderEvent)

	for _, event := range events {
		userID := c.extractAndValidateUserID(event)
		if userID == "" {
			log.Info().Msgf("order event missing user_id: %+v", event)
			continue
		}

		flattenedEvents := model.FlattenOrderPlacedEvent(event)
		userEvents[userID] = append(userEvents[userID], flattenedEvents...)
	}

	return userEvents
}

func (c *OrderConsumer) extractAndValidateUserID(event model.OrderPlacedEvent) string {
	userID := event.OrderPlacedEventData.UserID
	if userID == 0 {
		return ""
	}
	return strconv.FormatInt(userID, 10)
}

// persistUserEvents persists events for each user in parallel
func (c *OrderConsumer) persistUserEvents(userEvents map[string][]model.FlattenedOrderEvent) error {
	if len(userEvents) == 0 {
		return nil
	}

	var (
		mu     sync.Mutex
		errors []error
	)

	g := new(errgroup.Group)

	for id, userToEvents := range userEvents {
		userID := id
		events := userToEvents

		g.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Str("user_id", userID).
						Interface("panic", r).
						Msg("panic while persisting order events")
					mu.Lock()
					errors = append(errors, fmt.Errorf("panic for user %s: %v", userID, r))
					mu.Unlock()
				}
			}()

			if err := c.handler.Persist(userID, events); err != nil {
				log.Error().
					Str("user_id", userID).
					Err(err).
					Msg("failed to persist order events")
				mu.Lock()
				errors = append(errors, fmt.Errorf("persist failed for user %s: %w", userID, err))
				mu.Unlock()
			}

			return nil
		})
	}

	g.Wait()

	if len(errors) > 0 {
		var processingError error
		for _, err := range errors {
			if processingError == nil {
				processingError = err
			} else {
				processingError = fmt.Errorf("%v; %w", processingError, err)
			}
		}

		log.Error().
			Int("failed_users", len(errors)).
			Int("total_users", len(userEvents)).
			Msg("batch processing completed with errors")

		return processingError
	}

	log.Info().
		Int("user_count", len(userEvents)).
		Msg("successfully processed batch")

	return nil
}
