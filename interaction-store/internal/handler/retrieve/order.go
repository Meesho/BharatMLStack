package retrieve

import (
	"fmt"
	"sort"
	"strconv"
	"sync"

	blocks "github.com/Meesho/BharatMLStack/interaction-store/internal/data/block"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/enum"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/scylla"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
)

var (
	orderRetrieveHandler *OrderRetrieveHandler
	orderOnce            sync.Once
)

type OrderRetrieveHandler struct {
	scyllaDb scylla.Database
}

func InitOrderRetrieveHandler() *OrderRetrieveHandler {
	if orderRetrieveHandler == nil {
		orderOnce.Do(func() {
			orderRetrieveHandler = &OrderRetrieveHandler{
				scyllaDb: scylla.NewDatabase(),
			}
		})
	}
	return orderRetrieveHandler
}

func (o *OrderRetrieveHandler) Retrieve(userId string, startTimestampMs int64, endTimestampMs int64, limit int32) (data any, err error) {
	if limit > 2000 {
		limit = 2000
	}
	lowerBound := utils.WeekFromTimestampMs(startTimestampMs) % 24
	upperBound := utils.WeekFromTimestampMs(endTimestampMs) % 24
	tablesToFields := make(map[string][]string)
	if lowerBound <= upperBound {
		for i := lowerBound; i <= upperBound; i++ {
			bucketIdx := i / 8
			tablesToFields[getOrderTableName(bucketIdx)] = append(tablesToFields[getOrderTableName(bucketIdx)], "week_"+strconv.Itoa(i))
		}
	} else {
		for i := lowerBound; i < 24; i++ {
			bucketIdx := i / 8
			tablesToFields[getOrderTableName(bucketIdx)] = append(tablesToFields[getOrderTableName(bucketIdx)], "week_"+strconv.Itoa(i))
		}
		for i := 0; i <= upperBound; i++ {
			bucketIdx := i / 8
			tablesToFields[getOrderTableName(bucketIdx)] = append(tablesToFields[getOrderTableName(bucketIdx)], "week_"+strconv.Itoa(i))
		}
	}

	weekToData, err := o.fetchDataInParallel(tablesToFields, userId)
	if err != nil {
		return nil, err
	}

	weekToDeserializedBlocks, err := o.buildDeserialisedPermanentStorageDataBlocks(weekToData)
	if err != nil {
		return nil, err
	}

	allEvents := make([]model.FlatOrderEvent, 0)
	for _, ddb := range weekToDeserializedBlocks {
		events, err := ddb.RetrieveEventData()
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, events.([]model.FlatOrderEvent)...)
	}

	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].OrderedAt < allEvents[j].OrderedAt
	})

	filteredEvents := make([]model.FlatOrderEvent, 0)
	for _, event := range allEvents {
		orderedAt := event.OrderedAt
		if orderedAt >= startTimestampMs && orderedAt <= endTimestampMs {
			filteredEvents = append(filteredEvents, event)
		}
	}

	if limit > 0 && len(filteredEvents) > int(limit) {
		filteredEvents = filteredEvents[len(filteredEvents)-int(limit):]
	}

	return filteredEvents, nil
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

func (o *OrderRetrieveHandler) fetchDataInParallel(tablesToFields map[string][]string, userId string) (map[string][]byte, error) {
	type fetchResult struct {
		tableName string
		columns   []string
		data      map[string]interface{}
		err       error
	}

	weekToData := make(map[string][]byte)
	var wg sync.WaitGroup
	resultChan := make(chan fetchResult, len(tablesToFields))

	for tableName, fields := range tablesToFields {
		if len(fields) == 0 {
			continue
		}
		wg.Add(1)
		go func(tblName string, cols []string) {
			defer wg.Done()
			data, err := o.scyllaDb.RetrieveInteractions(tblName, userId, cols)
			if err != nil {
				resultChan <- fetchResult{
					tableName: tblName,
					columns:   cols,
					err:       fmt.Errorf("failed to retrieve data for table: %s, error: %w", tblName, err),
				}
				return
			}
			resultChan <- fetchResult{
				tableName: tblName,
				columns:   cols,
				data:      data,
			}
		}(tableName, fields)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.err != nil {
			return nil, result.err
		}
		for _, column := range result.columns {
			value, ok := result.data[column]
			if !ok {
				weekToData[column] = nil
				continue
			}
			rawData, ok := value.([]byte)
			if !ok {
				return nil, fmt.Errorf("unexpected data type for column %s in table %s: got %T, want []byte", column, result.tableName, value)
			}
			weekToData[column] = rawData
		}
	}

	return weekToData, nil
}

func (o *OrderRetrieveHandler) buildDeserialisedPermanentStorageDataBlocks(weekToData map[string][]byte) (map[string]*blocks.DeserializedPSDB, error) {
	weekToDeserializedBlocks := make(map[string]*blocks.DeserializedPSDB)
	for column, data := range weekToData {
		if data == nil {
			continue
		}
		ddb, err := blocks.DeserializePSDB(data, enum.InteractionTypeOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize data for column %s, error: %w", column, err)
		}
		weekToDeserializedBlocks[column] = ddb
	}
	return weekToDeserializedBlocks, nil
}
