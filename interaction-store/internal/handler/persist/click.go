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

const maxEventsPerWeek = 500

func (p *ClickPersistHandler) Persist(userId string, data any) error {
	clickEvents, ok := data.([]model.ClickEvent)
	if !ok {
		return fmt.Errorf("unexpected data type for persistence: got %T, want []ClickEvent", data)
	}

	bucketEvents, weekFlags := p.partitionEventsByBucket(clickEvents)
	for bucketIdx, events := range bucketEvents {
		if err := p.persistToBucket(bucketIdx, userId, events, weekFlags); err != nil {
			return err
		}
	}
	return nil
}

func (p *ClickPersistHandler) partitionEventsByBucket(events []model.ClickEvent) (map[int][]model.ClickEvent, []bool) {
	bucketEvents := make(map[int][]model.ClickEvent)
	weekFlags := make([]bool, 24)

	for _, event := range events {
		week := utils.WeekFromTimestampMs(event.ClickEventData.Payload.ClickedAt) % 24
		bucketIdx := week / 8
		bucketEvents[bucketIdx] = append(bucketEvents[bucketIdx], event)
		weekFlags[week] = true
	}
	return bucketEvents, weekFlags
}

func (p *ClickPersistHandler) persistToBucket(bucketIdx int, userId string, events []model.ClickEvent, weekFlags []bool) error {
	columns := p.getColumnsForBucket(bucketIdx, weekFlags)
	if len(columns) == 0 {
		return nil
	}

	existingData, err := p.scyllaDb.RetrieveInteractions(getTableName(bucketIdx), userId, columns)
	if err != nil {
		return fmt.Errorf("retrieve interactions failed for user %s: %w", userId, err)
	}

	storageBlocks, err := p.deserializeExistingData(existingData)
	if err != nil {
		return err
	}

	finalEvents, columnsToInsert, columnsToUpdate, err := p.mergeEvents(events, storageBlocks)
	if err != nil {
		return err
	}

	metadata, err := p.persistEvents(bucketIdx, userId, finalEvents, columnsToInsert, columnsToUpdate)
	if err != nil {
		return err
	}

	go p.updateMetadata(userId, metadata.toUpdate, metadata.toInsert)
	return nil
}

func (p *ClickPersistHandler) getColumnsForBucket(bucketIdx int, weekFlags []bool) []string {
	var columns []string
	startWeek, endWeek := bucketIdx*8, (bucketIdx+1)*8
	for week := startWeek; week < endWeek; week++ {
		if weekFlags[week] {
			columns = append(columns, "week_"+strconv.Itoa(week))
		}
	}
	return columns
}

func (p *ClickPersistHandler) deserializeExistingData(data map[string]interface{}) (map[string]*blocks.DeserializedPSDB, error) {
	result := make(map[string]*blocks.DeserializedPSDB)
	for column, value := range data {
		ddb, err := blocks.DeserializePSDB(value.([]byte), enum.InteractionTypeClick)
		if err != nil {
			return nil, fmt.Errorf("deserialize failed for column %s: %w", column, err)
		}
		result[column] = ddb
	}
	return result, nil
}

func (p *ClickPersistHandler) mergeEvents(
	newEvents []model.ClickEvent,
	storageBlocks map[string]*blocks.DeserializedPSDB,
) (map[string][]model.ClickEvent, []string, []string, error) {
	finalEvents := make(map[string][]model.ClickEvent)
	var columnsToInsert, columnsToUpdate []string

	for _, event := range newEvents {
		week := utils.WeekFromTimestampMs(event.ClickEventData.Payload.ClickedAt) % 24
		column := "week_" + strconv.Itoa(week)

		ddb, ok := storageBlocks[column]
		if !ok {
			return nil, nil, nil, fmt.Errorf("storage block not found for column: %s", column)
		}

		existing, err := p.getExistingClickEvents(ddb, column)
		if err != nil {
			return nil, nil, nil, err
		}

		if len(existing) == 0 {
			columnsToInsert = append(columnsToInsert, column)
		} else {
			columnsToUpdate = append(columnsToUpdate, column)
		}

		merged := p.mergeAndTrimEvents(existing, event)
		finalEvents[column] = merged
	}

	return finalEvents, columnsToInsert, columnsToUpdate, nil
}

func (p *ClickPersistHandler) getExistingClickEvents(ddb *blocks.DeserializedPSDB, column string) ([]model.ClickEvent, error) {
	data, err := ddb.RetrieveEventData()
	if err != nil {
		return nil, fmt.Errorf("retrieve event data failed for column %s: %w", column, err)
	}
	events, ok := data.([]model.ClickEvent)
	if !ok {
		return nil, fmt.Errorf("unexpected data type in deserialized block for column: %s", column)
	}
	return events, nil
}

func (p *ClickPersistHandler) mergeAndTrimEvents(existing []model.ClickEvent, newEvent model.ClickEvent) []model.ClickEvent {
	if len(existing) > 0 {
		lastTimestamp := existing[len(existing)-1].ClickEventData.Payload.ClickedAt
		if utils.TimestampDiffInWeeks(newEvent.ClickEventData.Payload.ClickedAt, lastTimestamp) >= 24 {
			existing = existing[:0]
		}
	}

	existing = append(existing, newEvent)
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].ClickEventData.Payload.ClickedAt < existing[j].ClickEventData.Payload.ClickedAt
	})

	if len(existing) > maxEventsPerWeek {
		existing = existing[1:]
	}
	return existing
}

type metadataMaps struct {
	toUpdate map[string]interface{}
	toInsert map[string]interface{}
}

func (p *ClickPersistHandler) persistEvents(
	bucketIdx int,
	userId string,
	finalEvents map[string][]model.ClickEvent,
	columnsToInsert, columnsToUpdate []string,
) (*metadataMaps, error) {
	metadata := &metadataMaps{
		toUpdate: make(map[string]interface{}),
		toInsert: make(map[string]interface{}),
	}
	tableName := getTableName(bucketIdx)

	for _, column := range columnsToUpdate {
		events := finalEvents[column]
		data, err := p.buildAndSerializePermanentStorageDataBlock(events)
		if err != nil {
			return nil, fmt.Errorf("serialize failed for column %s: %w", column, err)
		}
		if err := p.scyllaDb.UpdateInteractions(tableName, userId, column, data); err != nil {
			return nil, fmt.Errorf("update failed for column %s: %w", column, err)
		}
		metadata.toUpdate[column] = len(events)
	}

	if len(columnsToInsert) > 0 {
		insertData := make(map[string]interface{})
		for _, column := range columnsToInsert {
			events := finalEvents[column]
			data, err := p.buildAndSerializePermanentStorageDataBlock(events)
			if err != nil {
				return nil, fmt.Errorf("serialize failed for column %s: %w", column, err)
			}
			insertData[column] = data
			metadata.toInsert[column] = len(events)
		}
		if err := p.scyllaDb.PersistInteractions(tableName, userId, insertData); err != nil {
			return nil, fmt.Errorf("persist failed for columns %v: %w", columnsToInsert, err)
		}
	}

	return metadata, nil
}

func (p *ClickPersistHandler) updateMetadata(userId string, metadataToUpdate, metadataToInsert map[string]interface{}) {
	for column, count := range metadataToUpdate {
		_ = p.scyllaDb.UpdateMetadata(getMetadataTableName(), userId, column, count)
	}

	if len(metadataToInsert) > 0 {
		_ = p.scyllaDb.PersistMetadata(getMetadataTableName(), userId, metadataToInsert)
	}
}

func getMetadataTableName() string {
	return "click_interactions_metadata"
}

func getTableName(count int) string {
	switch count {
	case 0:
		return "click_events_bucket1"
	case 1:
		return "click_events_bucket2"
	case 2:
		return "click_events_bucket3"
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
