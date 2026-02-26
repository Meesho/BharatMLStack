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
	orderPersistHandler *OrderPersistHandler
	orderOnce           sync.Once
)

type OrderPersistHandler struct {
	scyllaDb scylla.Database
}

func InitOrderPersistHandler() *OrderPersistHandler {
	if orderPersistHandler == nil {
		orderOnce.Do(func() {
			orderPersistHandler = &OrderPersistHandler{
				scyllaDb: scylla.NewDatabase(),
			}
		})
	}
	return orderPersistHandler
}

func (p *OrderPersistHandler) Persist(userId string, data []model.FlattenedOrderEvent) error {
	bucketEvents, weekFlags := p.partitionEventsByBucket(data)
	for bucketIdx, events := range bucketEvents {
		if err := p.persistToBucket(bucketIdx, userId, events, weekFlags); err != nil {
			return err
		}
	}
	return nil
}

func (p *OrderPersistHandler) partitionEventsByBucket(events []model.FlattenedOrderEvent) (map[int][]model.FlattenedOrderEvent, []bool) {
	bucketEvents := make(map[int][]model.FlattenedOrderEvent)
	weekFlags := make([]bool, constants.TotalWeeks)

	for _, event := range events {
		week := utils.WeekFromTimestampMs(event.OrderedAt) % constants.TotalWeeks
		bucketIdx := week / constants.WeeksPerBucket
		bucketEvents[bucketIdx] = append(bucketEvents[bucketIdx], event)
		weekFlags[week] = true
	}
	return bucketEvents, weekFlags
}

func (p *OrderPersistHandler) persistToBucket(bucketIdx int, userId string, events []model.FlattenedOrderEvent, weekFlags []bool) error {
	columns := p.getColumnsForBucket(bucketIdx, weekFlags)
	if len(columns) == 0 {
		return nil
	}

	existingData, err := p.scyllaDb.RetrieveInteractions(getOrderTableName(bucketIdx), userId, columns)
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

func (p *OrderPersistHandler) getColumnsForBucket(bucketIdx int, weekFlags []bool) []string {
	var columns []string
	startWeek, endWeek := bucketIdx*constants.WeeksPerBucket, (bucketIdx+1)*constants.WeeksPerBucket
	for week := endWeek - 1; week >= startWeek; week-- {
		if weekFlags[week] {
			columns = append(columns, constants.WeekColumnPrefix+strconv.Itoa(week))
		}
	}
	return columns
}

func (p *OrderPersistHandler) deserializeExistingData(data map[string]interface{}) (map[string]*blocks.DeserializedPSDB, error) {
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
		ddb, err := blocks.DeserializePSDB(byteVal, enum.InteractionTypeOrder)
		if err != nil {
			return nil, fmt.Errorf("deserialize failed for column %s: %w", column, err)
		}
		result[column] = ddb
	}
	return result, nil
}

func (p *OrderPersistHandler) mergeEvents(newEvents []model.FlattenedOrderEvent, storageBlocks map[string]*blocks.DeserializedPSDB, userId string) (map[string][]model.FlattenedOrderEvent, error) {
	finalEvents := make(map[string][]model.FlattenedOrderEvent)

	for _, event := range newEvents {
		week := utils.WeekFromTimestampMs(event.OrderedAt) % constants.TotalWeeks
		column := constants.WeekColumnPrefix + strconv.Itoa(week)

		accumulated, alreadyProcessed := finalEvents[column]
		if !alreadyProcessed {
			ddb, existsInStorage := storageBlocks[column]
			if existsInStorage {
				existing, err := p.getExistingOrderEvents(ddb, column, userId)
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

func (p *OrderPersistHandler) getExistingOrderEvents(ddb *blocks.DeserializedPSDB, column string, userId string) ([]model.FlattenedOrderEvent, error) {
	data, err := ddb.RetrieveEventData(userId)
	if err != nil {
		return nil, fmt.Errorf("retrieve event data failed for column %s: %w", column, err)
	}
	events, ok := data.([]model.FlattenedOrderEvent)
	if !ok {
		return nil, fmt.Errorf("unexpected data type in deserialized block for column: %s", column)
	}
	return events, nil
}

func (p *OrderPersistHandler) mergeAndTrimEvents(existing []model.FlattenedOrderEvent, newEvent model.FlattenedOrderEvent) []model.FlattenedOrderEvent {
	if len(existing) > 0 {
		largestTimestamp := existing[0].OrderedAt
		if utils.TimestampDiffInWeeks(newEvent.OrderedAt, largestTimestamp) >= constants.TotalWeeks {
			existing = existing[:0]
		}
	}

	existing = append(existing, newEvent)
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].OrderedAt > existing[j].OrderedAt
	})

	if len(existing) > constants.MaxOrderEventsPerWeek {
		existing = existing[:constants.MaxOrderEventsPerWeek]
	}
	return existing
}

func (p *OrderPersistHandler) persistEvents(bucketIdx int, userId string, finalEvents map[string][]model.FlattenedOrderEvent) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	tableName := getOrderTableName(bucketIdx)

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

func (p *OrderPersistHandler) updateMetadata(userId string, metadataToUpdate map[string]interface{}) error {
	if len(metadataToUpdate) == 0 {
		return nil
	}

	tableName := getOrderMetadataTableName()
	if err := p.scyllaDb.UpdateMetadata(tableName, userId, metadataToUpdate); err != nil {
		return fmt.Errorf("failed to update order metadata for userId=%s, table=%s: %w", userId, tableName, err)
	}
	return nil
}

func getOrderMetadataTableName() string {
	return "order_interactions_metadata"
}

func getOrderTableName(bucketIdx int) string {
	switch bucketIdx {
	case 0:
		return "order_interactions_bucket1"
	case 1:
		return "order_interactions_bucket2"
	case 2:
		return "order_interactions_bucket3"
	}
	return ""
}

func (p *OrderPersistHandler) buildPermanentStorageDataBlock(data []model.FlattenedOrderEvent) (*blocks.PermanentStorageDataBlock, error) {
	builder := blocks.NewPermanentStorageDataBlockBuilder()
	builder.SetLayoutVersion(1)
	builder.SetCompressionType(compression.TypeZSTD)
	builder.SetData(data)
	builder.SetDataLength(uint16(len(data)))
	builder.SetInteractionType(enum.InteractionTypeOrder)
	return builder.Build(), nil
}
