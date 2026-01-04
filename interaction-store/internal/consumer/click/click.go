package click

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Meesho/interaction-store/internal/data/model"
	"github.com/Meesho/interaction-store/internal/handler/persist"
	"github.com/rs/zerolog/log"
)

var (
	clickConsumer Consumer
	clickOnce     sync.Once
)

type ClickConsumer struct {
	handler persist.PersistHandler
}

func newClickConsumer() Consumer {
	if clickConsumer == nil {
		clickOnce.Do(func() {
			clickConsumer = &ClickConsumer{
				handler: persist.InitClickPersistHandler(),
			}
		})
	}
	return clickConsumer
}

func (c *ClickConsumer) Process(events []model.ClickEvent) error {
	userEvents := make(map[string][]model.ClickEvent)
	for _, event := range events {
		userID := event.ClickEventData.Payload.UserId
		if strings.TrimSpace(userID) == "" {
			userID = event.ClickEventData.Payload.AnonymousUserId
		}
		if strings.TrimSpace(userID) != "" {
			userEvents[userID] = append(userEvents[userID], event)
		} else {
			log.Error().Msgf("event missing both user_id and anonymous_user_id: %v", event)
		}
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
		log.Error().Err(processingError).Msg("error processing user click events")
		return processingError
	}

	return nil
}
