package persist

import (
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/compression"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/constants"
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

func (p *ClickPersistHandler) Persist(userId string, data []model.ClickEvent) error {
	bucketEvents, weekFlags := p.partitionEventsByBucket(data)
	for bucketIdx, events := range bucketEvents {
		if err := p.persistToBucket(bucketIdx, userId, events, weekFlags); err != nil {
			return err
		}
	}
	return nil
}

func (p *ClickPersistHandler) partitionEventsByBucket(events []model.ClickEvent) (map[int][]model.ClickEvent, []bool) {
	bucketToEvents := make(map[int][]model.ClickEvent)
	weekFlags := make([]bool, constants.TotalWeeks)

	for _, event := range events {
		week := utils.WeekFromTimestampMs(event.ClickEventData.Payload.ClickedAt) % constants.TotalWeeks
		bucketIdx := week / constants.WeeksPerBucket
		bucketToEvents[bucketIdx] = append(bucketToEvents[bucketIdx], event)
		weekFlags[week] = true
	}
	return bucketToEvents, weekFlags
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

	finalEvents, err := p.mergeEvents(events, storageBlocks, userId)
	if err != nil {
		return err
	}

	metadata, err := p.persistEvents(bucketIdx, userId, finalEvents)
	if err != nil {
		return err
	}

	if err := p.updateMetadata(userId, metadata); err != nil {
		return err
	}
	return nil
}

func (p *ClickPersistHandler) getColumnsForBucket(bucketIdx int, weekFlags []bool) []string {
	var columns []string
	startWeek, endWeek := bucketIdx*constants.WeeksPerBucket, (bucketIdx+1)*constants.WeeksPerBucket
	for week := endWeek - 1; week >= startWeek; week-- {
		if weekFlags[week] {
			columns = append(columns, constants.WeekColumnPrefix+strconv.Itoa(week))
		}
	}
	return columns
}

func (p *ClickPersistHandler) deserializeExistingData(data map[string]interface{}) (map[string]*blocks.DeserializedPSDB, error) {
	result := make(map[string]*blocks.DeserializedPSDB)
	for column, value := range data {
		if value == nil {
			continue
		}
		byteVal, ok := value.([]byte)
		if !ok {
			return nil, fmt.Errorf("unexpected type for column %s: got %T, want []byte", column, value)
		}
		// Skip empty byte slices (new user case)
		if len(byteVal) == 0 {
			continue
		}
		ddb, err := blocks.DeserializePSDB(byteVal, enum.InteractionTypeClick)
		if err != nil {
			return nil, fmt.Errorf("deserialize failed for column %s: %w", column, err)
		}
		result[column] = ddb
	}
	return result, nil
}

func (p *ClickPersistHandler) mergeEvents(newEvents []model.ClickEvent, storageBlocks map[string]*blocks.DeserializedPSDB, userId string) (map[string][]model.ClickEvent, error) {
	finalEvents := make(map[string][]model.ClickEvent)

	for _, event := range newEvents {
		week := utils.WeekFromTimestampMs(event.ClickEventData.Payload.ClickedAt) % constants.TotalWeeks
		column := constants.WeekColumnPrefix + strconv.Itoa(week)

		accumulated, alreadyProcessed := finalEvents[column]
		if !alreadyProcessed {
			ddb, existsInStorage := storageBlocks[column]
			if existsInStorage {
				existing, err := p.getExistingClickEvents(ddb, column, userId)
				if err != nil {
					return nil, err
				}
				accumulated = existing
			}
		}

		merged := p.mergeAndTrimEvents(accumulated, event)
		finalEvents[column] = merged
	}

	return finalEvents, nil
}

func (p *ClickPersistHandler) getExistingClickEvents(ddb *blocks.DeserializedPSDB, column string, userId string) ([]model.ClickEvent, error) {
	data, err := ddb.RetrieveEventData(userId)
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
		largestTimestamp := existing[0].ClickEventData.Payload.ClickedAt
		if utils.TimestampDiffInWeeks(newEvent.ClickEventData.Payload.ClickedAt, largestTimestamp) >= constants.TotalWeeks {
			existing = existing[:0]
		}
	}

	existing = append(existing, newEvent)
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].ClickEventData.Payload.ClickedAt > existing[j].ClickEventData.Payload.ClickedAt
	})

	if len(existing) > constants.MaxClickEventsPerWeek {
		existing = existing[:constants.MaxClickEventsPerWeek]
	}
	return existing
}

func (p *ClickPersistHandler) persistEvents(bucketIdx int, userId string, finalEvents map[string][]model.ClickEvent) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	tableName := getTableName(bucketIdx)

	// Track all PSDBs for cleanup after persist completes
	psdbsToCleanup := make([]*blocks.PermanentStorageDataBlock, 0, len(finalEvents))

	if len(finalEvents) > 0 {
		updateData := make(map[string]interface{})
		columnPSDBs := make(map[string]*blocks.PermanentStorageDataBlock, len(finalEvents))

		// Build all PSDBs
		for column, events := range finalEvents {
			psdb, err := p.buildPermanentStorageDataBlock(events)
			if err != nil {
				cleanupPSDBs(psdbsToCleanup)
				return nil, fmt.Errorf("psdb build failed for column %s: %w", column, err)
			}
			columnPSDBs[column] = psdb
			psdbsToCleanup = append(psdbsToCleanup, psdb)
			metadata[column] = len(events)
		}

		// Serialize all PSDBs
		for column := range finalEvents {
			psdb := columnPSDBs[column]
			data, err := psdb.Serialize()
			if err != nil {
				cleanupPSDBs(psdbsToCleanup)
				return nil, fmt.Errorf("psdb serialize failed for column %s: %w", column, err)
			}
			updateData[column] = data
		}

		// Update all at once
		if err := p.scyllaDb.UpdateInteractions(tableName, userId, updateData); err != nil {
			cleanupPSDBs(psdbsToCleanup)
			return nil, fmt.Errorf("psdb update failed: %w", err)
		}
	}

	// Cleanup PSDBs after all persist operations complete
	cleanupPSDBs(psdbsToCleanup)

	return metadata, nil
}

func (p *ClickPersistHandler) updateMetadata(userId string, metadataToUpdate map[string]interface{}) error {
	if len(metadataToUpdate) == 0 {
		return nil
	}

	tableName := getMetadataTableName()
	if err := p.scyllaDb.UpdateMetadata(tableName, userId, metadataToUpdate); err != nil {
		return fmt.Errorf("failed to update click metadata for userId=%s, table=%s: %w", userId, tableName, err)
	}
	return nil
}

func getMetadataTableName() string {
	return "click_interactions_metadata"
}

func getTableName(bucketIdx int) string {
	switch bucketIdx {
	case 0:
		return "click_interactions_bucket1"
	case 1:
		return "click_interactions_bucket2"
	case 2:
		return "click_interactions_bucket3"
	}
	return ""
}

func (p *ClickPersistHandler) buildPermanentStorageDataBlock(data []model.ClickEvent) (*blocks.PermanentStorageDataBlock, error) {
	builder := blocks.NewPermanentStorageDataBlockBuilder()
	builder.SetLayoutVersion(1)
	builder.SetCompressionType(compression.TypeZSTD)
	builder.SetData(data)
	builder.SetDataLength(uint16(len(data)))
	builder.SetInteractionType(enum.InteractionTypeClick)
	return builder.Build(), nil
}

// cleanupPSDBs returns all PSDBs to the pool after use
func cleanupPSDBs(psdbs []*blocks.PermanentStorageDataBlock) {
	psdbPool := blocks.GetPSDBPool()
	for _, psdb := range psdbs {
		if psdb != nil {
			psdbPool.Put(psdb)
		}
	}
}
