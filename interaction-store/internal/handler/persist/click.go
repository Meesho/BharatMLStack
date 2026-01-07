package persist

import (
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/compression"
	blocks "github.com/Meesho/BharatMLStack/interaction-store/internal/data/block"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/enum"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/scylla"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
)

var (
	clickPersistHandler *ClickPersistHandler
	clickOnce           sync.Once
)

type ClickPersistHandler struct {
	scyllaDb scylla.Database
}

func InitClickPersistHandler() *ClickPersistHandler {
	if clickPersistHandler == nil {
		clickOnce.Do(func() {
			clickPersistHandler = &ClickPersistHandler{
				scyllaDb: scylla.NewDatabase(),
			}
		})
	}
	return clickPersistHandler
}

func (p *ClickPersistHandler) Persist(userId string, data any) error {
	clickEvents, ok := data.([]model.ClickEvent)
	if !ok {
		return fmt.Errorf("unexpected data type for persistence: got %T, want []ClickEvent", data)
	}
	tablesToEvents := make(map[int][]model.ClickEvent)
	weeks := make([]bool, 24)
	for _, clickEvent := range clickEvents {
		clickedAt := clickEvent.ClickEventData.Payload.ClickedAt
		week := utils.WeekFromTimestampMs(clickedAt) % 24
		if week >= 0 && week <= 7 {
			tablesToEvents[0] = append(tablesToEvents[0], clickEvent)
			weeks[week] = true
		} else if week >= 8 && week <= 15 {
			tablesToEvents[1] = append(tablesToEvents[1], clickEvent)
			weeks[week] = true
		} else if week >= 16 && week <= 23 {
			tablesToEvents[2] = append(tablesToEvents[2], clickEvent)
			weeks[week] = true
		}
	}
	for i, events := range tablesToEvents {
		err := p.persistToDb(weeks, i, userId, events)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ClickPersistHandler) persistToDb(weeks []bool, count int, userId string, events []model.ClickEvent) error {
	columns := []string{}
	for i := count * 8; i < (count+1)*8; i++ {
		if weeks[i] {
			columns = append(columns, "week_"+strconv.Itoa(i))
		}
	}
	if len(columns) == 0 {
		return nil
	}

	existingData, err := p.scyllaDb.RetrieveInteractions(getTableName(count), userId, columns)
	if err != nil {
		return fmt.Errorf("failed to execute retrieve query for: %s, interaction type: CLICK, error: %w", userId, err)
	}

	storageBlocksInitial := make(map[string]*blocks.DeserializedPSDB)
	finalEvents := make(map[string][]model.ClickEvent)
	columnsToInsert := []string{}
	columnsToUpdate := []string{}
	for column, value := range existingData {
		ddb, err := blocks.DeserializePSDB(value.([]byte), enum.InteractionTypeClick)
		if err != nil {
			return err
		}
		storageBlocksInitial[column] = ddb
	}

	for _, event := range events {
		week := utils.WeekFromTimestampMs(event.ClickEventData.Payload.ClickedAt) % 24
		field := "week_" + strconv.Itoa(week)
		ddb, ok := storageBlocksInitial[field]
		if !ok {
			return fmt.Errorf("failed to get storage block for field: %s", field)
		}
		existingEvents, err := ddb.RetrieveEventData()
		if err != nil {
			return fmt.Errorf("failed to retrieve event data for field: %s, error: %w", field, err)
		}
		clickEvents, ok := existingEvents.([]model.ClickEvent)
		if !ok {
			return fmt.Errorf("unexpected data type in deserialized block for field: %s", field)
		}
		if len(clickEvents) == 0 {
			columnsToInsert = append(columnsToInsert, field)
		} else {
			columnsToUpdate = append(columnsToUpdate, field)
		}
		if utils.TimestampDiffInWeeks(event.ClickEventData.Payload.ClickedAt,
			clickEvents[len(clickEvents)-1].ClickEventData.Payload.ClickedAt) >= 24 {
			clickEvents = clickEvents[:0]
		}
		clickEvents = append(clickEvents, event)
		sort.Slice(existingEvents, func(i, j int) bool {
			return clickEvents[i].ClickEventData.Payload.ClickedAt < clickEvents[j].ClickEventData.Payload.ClickedAt
		})
		if len(clickEvents) > 500 {
			clickEvents = clickEvents[1:]
		}
		finalEvents[field] = clickEvents
	}

	for _, column := range columnsToUpdate {
		events := finalEvents[column]
		data, err := p.buildAndSerializePermanentStorageDataBlock(events)
		if err != nil {
			return fmt.Errorf("failed to serialize permanent storage data block for column: %s, error: %w", column, err)
		}
		err = p.scyllaDb.UpdateInteractions(getTableName(count), userId, column, data)
		if err != nil {
			return fmt.Errorf("failed to update data for column: %s, error: %w", column, err)
		}
	}

	columnsToDataInsert := make(map[string]interface{})
	for _, column := range columnsToInsert {
		events := finalEvents[column]
		data, err := p.buildAndSerializePermanentStorageDataBlock(events)
		if err != nil {
			return fmt.Errorf("failed to serialize permanent storage data block for column: %s, error: %w", column, err)
		}
		columnsToDataInsert[column] = data
	}
	err = p.scyllaDb.PersistInteractions(getTableName(count), userId, columnsToDataInsert)
	if err != nil {
		return fmt.Errorf("failed to persist data for columns: %v, error: %w", columnsToInsert, err)
	}

	return nil
}

func getTableName(count int) string {
	switch count {
	case 0:
		return "click_events_week_0_7"
	case 1:
		return "click_events_week_8_15"
	case 2:
		return "click_events_week_16_23"
	}
	return ""
}

func (p *ClickPersistHandler) buildAndSerializePermanentStorageDataBlock(data []model.ClickEvent) ([]byte, error) {
	builder := blocks.NewPermanentStorageDataBlockBuilder()
	builder.SetLayoutVersion(1)
	builder.SetCompressionType(compression.TypeZSTD)
	builder.SetData(data)
	builder.SetDataLength(uint16(len(data)))
	builder.SetInteractionType(enum.InteractionTypeClick)
	psdb := builder.Build()
	return psdb.Serialize()
}
