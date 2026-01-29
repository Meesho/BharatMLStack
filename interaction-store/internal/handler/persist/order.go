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
	"github.com/rs/zerolog/log"
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

const maxOrderEventsPerWeek = 500

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
	weekFlags := make([]bool, 24)

	for _, event := range events {
		week := utils.WeekFromTimestampMs(event.OrderedAt) % 24
		bucketIdx := week / 8
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

	finalEvents, columnsToInsert, columnsToUpdate, err := p.mergeEvents(events, storageBlocks)
	if err != nil {
		return err
	}

	metadata, err := p.persistEvents(bucketIdx, userId, finalEvents, columnsToInsert, columnsToUpdate)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("panic occurred while updating order metadata for user %s: %v", userId, r)
			}
		}()
		p.updateMetadata(userId, metadata.toUpdate, metadata.toInsert)
	}()
	return nil
}

func (p *OrderPersistHandler) getColumnsForBucket(bucketIdx int, weekFlags []bool) []string {
	var columns []string
	startWeek, endWeek := bucketIdx*8, (bucketIdx+1)*8
	for week := startWeek; week < endWeek; week++ {
		if weekFlags[week] {
			columns = append(columns, "week_"+strconv.Itoa(week))
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

func (p *OrderPersistHandler) mergeEvents(newEvents []model.FlattenedOrderEvent, storageBlocks map[string]*blocks.DeserializedPSDB) (map[string][]model.FlattenedOrderEvent, []string, []string, error) {
	finalEvents := make(map[string][]model.FlattenedOrderEvent)
	var columnsToInsert, columnsToUpdate []string

	for _, event := range newEvents {
		week := utils.WeekFromTimestampMs(event.OrderedAt) % 24
		column := "week_" + strconv.Itoa(week)

		accumulated, alreadyProcessed := finalEvents[column]
		if !alreadyProcessed {
			ddb, existsInStorage := storageBlocks[column]
			if existsInStorage {
				existing, err := p.getExistingOrderEvents(ddb, column)
				if err != nil {
					return nil, nil, nil, err
				}
				accumulated = existing
				columnsToUpdate = append(columnsToUpdate, column)
			} else {
				columnsToInsert = append(columnsToInsert, column)
			}
		}

		merged := p.mergeAndTrimEvents(accumulated, event)
		finalEvents[column] = merged
	}

	return finalEvents, columnsToInsert, columnsToUpdate, nil
}

func (p *OrderPersistHandler) getExistingOrderEvents(ddb *blocks.DeserializedPSDB, column string) ([]model.FlattenedOrderEvent, error) {
	data, err := ddb.RetrieveEventData()
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
		if utils.TimestampDiffInWeeks(newEvent.OrderedAt, largestTimestamp) >= 24 {
			existing = existing[:0]
		}
	}

	existing = append(existing, newEvent)
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].OrderedAt > existing[j].OrderedAt
	})

	if len(existing) > maxOrderEventsPerWeek {
		existing = existing[:maxOrderEventsPerWeek]
	}
	return existing
}

func (p *OrderPersistHandler) persistEvents(bucketIdx int, userId string, finalEvents map[string][]model.FlattenedOrderEvent, columnsToInsert, columnsToUpdate []string) (*metadataMaps, error) {
	metadata := &metadataMaps{
		toUpdate: make(map[string]interface{}),
		toInsert: make(map[string]interface{}),
	}
	tableName := getOrderTableName(bucketIdx)

	// Track all PSDBs for cleanup after persist completes
	psdbsToCleanup := make([]*blocks.PermanentStorageDataBlock, 0, len(columnsToUpdate)+len(columnsToInsert))

	// Process updates: serialize and persist one at a time
	for _, column := range columnsToUpdate {
		events := finalEvents[column]
		psdb, err := p.buildPermanentStorageDataBlock(events)
		if err != nil {
			cleanupPSDBs(psdbsToCleanup)
			return nil, fmt.Errorf("psdb build failed for column %s: %w", column, err)
		}
		psdbsToCleanup = append(psdbsToCleanup, psdb)

		data, err := psdb.Serialize()
		if err != nil {
			cleanupPSDBs(psdbsToCleanup)
			return nil, fmt.Errorf("psdb serialize failed for column %s: %w", column, err)
		}
		if err := p.scyllaDb.UpdateInteractions(tableName, userId, column, data); err != nil {
			cleanupPSDBs(psdbsToCleanup)
			return nil, fmt.Errorf("psdb update failed for column %s: %w", column, err)
		}
		metadata.toUpdate[column] = len(events)
	}

	// Process inserts: build all PSDBs first, then serialize and persist together
	if len(columnsToInsert) > 0 {
		insertData := make(map[string]interface{})
		columnPSDBs := make(map[string]*blocks.PermanentStorageDataBlock, len(columnsToInsert))

		// Build all PSDBs
		for _, column := range columnsToInsert {
			events := finalEvents[column]
			psdb, err := p.buildPermanentStorageDataBlock(events)
			if err != nil {
				cleanupPSDBs(psdbsToCleanup)
				return nil, fmt.Errorf("psdb build failed for column %s: %w", column, err)
			}
			columnPSDBs[column] = psdb
			psdbsToCleanup = append(psdbsToCleanup, psdb)
			metadata.toInsert[column] = len(events)
		}

		// Serialize all PSDBs
		for _, column := range columnsToInsert {
			psdb := columnPSDBs[column]
			data, err := psdb.Serialize()
			if err != nil {
				cleanupPSDBs(psdbsToCleanup)
				return nil, fmt.Errorf("psdb serialize failed for column %s: %w", column, err)
			}
			insertData[column] = data
		}

		// Persist all at once
		if err := p.scyllaDb.PersistInteractions(tableName, userId, insertData); err != nil {
			cleanupPSDBs(psdbsToCleanup)
			return nil, fmt.Errorf("psdb persist failed for columns %v: %w", columnsToInsert, err)
		}
	}

	// Cleanup PSDBs after all persist operations complete
	cleanupPSDBs(psdbsToCleanup)

	return metadata, nil
}

func (p *OrderPersistHandler) updateMetadata(userId string, metadataToUpdate, metadataToInsert map[string]interface{}) {
	tableName := getOrderMetadataTableName()

	for column, count := range metadataToUpdate {
		if err := p.scyllaDb.UpdateMetadata(tableName, userId, column, count); err != nil {
			log.Error().Msg(fmt.Sprintf("failed to update order metadata for userId=%s, table=%s, column=%s: %v", userId, tableName, column, err))
		}
	}

	if len(metadataToInsert) > 0 {
		if err := p.scyllaDb.PersistMetadata(tableName, userId, metadataToInsert); err != nil {
			log.Error().Msg(fmt.Sprintf("failed to persist order metadata for userId=%s, table=%s, columns=%v: %v", userId, tableName, metadataToInsert, err))
		}
	}
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
