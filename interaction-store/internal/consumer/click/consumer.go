package click

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/handler/persist"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Consumer interface {
	Process(events []model.ClickEvent) error
}

var (
	clickConsumer Consumer
	syncOnce      sync.Once
)

type ClickConsumer struct {
	handler persist.PersistHandler[[]model.ClickEvent]
}

func newClickConsumer() Consumer {
	if clickConsumer == nil {
		syncOnce.Do(func() {
			clickConsumer = &ClickConsumer{
				handler: persist.InitClickPersistHandler(),
			}
		})
	}
	return clickConsumer
}

func (c *ClickConsumer) Process(events []model.ClickEvent) error {
	userEvents := c.preprocessAndValidateEvents(events)
	return c.persistUserEvents(userEvents)
}

// preprocessAndValidateEvents validates and groups click events by user ID
func (c *ClickConsumer) preprocessAndValidateEvents(events []model.ClickEvent) map[string][]model.ClickEvent {
	userEvents := make(map[string][]model.ClickEvent)

	for _, event := range events {
		userID := c.extractUserID(event)
		if userID == "" {
			log.Error().Msgf("event missing both user_id and anonymous_user_id: %v", event)
			continue
		}
		userEvents[userID] = append(userEvents[userID], event)
	}

	return userEvents
}

// extractUserID extracts and validates user ID from click event
// Returns user_id if present, otherwise anonymous_user_id, or empty string if both missing
func (c *ClickConsumer) extractUserID(event model.ClickEvent) string {
	userID := strings.TrimSpace(event.ClickEventData.Payload.UserId)
	if userID != "" {
		return userID
	}
	anonymousUserID := strings.TrimSpace(event.ClickEventData.Payload.AnonymousUserId)
	return anonymousUserID
}

// persistUserEvents persists events for each user in parallel
func (c *ClickConsumer) persistUserEvents(userEvents map[string][]model.ClickEvent) error {
	if len(userEvents) == 0 {
		return nil
	}

	var (
		mu     sync.Mutex
		errors []error
	)

	g := new(errgroup.Group)

	for id, userScopedEvents := range userEvents {
		userID := id
		events := userScopedEvents

		g.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Str("user_id", userID).
						Interface("panic", r).
						Msg("panic while persisting click events")
					mu.Lock()
					errors = append(errors, fmt.Errorf("panic for user %s: %v", userID, r))
					mu.Unlock()
				}
			}()

			if err := c.handler.Persist(userID, events); err != nil {
				log.Error().
					Str("user_id", userID).
					Err(err).
					Msg("failed to persist click events")
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
