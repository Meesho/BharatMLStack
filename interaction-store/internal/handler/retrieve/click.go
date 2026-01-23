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
	"github.com/rs/zerolog/log"
)

var (
	clickRetrieveHandler *ClickRetrieveHandler
	clickOnce            sync.Once
)

type ClickRetrieveHandler struct {
	scyllaDb scylla.Database
}

func InitClickRetrieveHandler() *ClickRetrieveHandler {
	if clickRetrieveHandler == nil {
		clickOnce.Do(func() {
			clickRetrieveHandler = &ClickRetrieveHandler{
				scyllaDb: scylla.NewDatabase(),
			}
		})
	}
	return clickRetrieveHandler
}

func (c *ClickRetrieveHandler) Retrieve(userId string, startTimestampMs int64, endTimestampMs int64, limit int32) ([]model.ClickEvent, error) {
	if err := validateTimeRange(startTimestampMs, endTimestampMs); err != nil {
		return nil, err
	}

	limit = capLimit(limit, 2000)
	tablesToFields := c.buildTableToFieldsMapping(startTimestampMs, endTimestampMs, getTableName)

	weekToData, err := c.fetchDataInParallel(tablesToFields, userId)
	if err != nil {
		return nil, err
	}

	weekToDeserializedBlocks, err := c.buildDeserialisedPermanentStorageDataBlocks(weekToData)
	if err != nil {
		return nil, err
	}

	allEvents := make([]model.ClickEvent, 0)
	for _, ddb := range weekToDeserializedBlocks {
		events, err := ddb.RetrieveEventData()
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, events.([]model.ClickEvent)...)
	}

	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].ClickEventData.Payload.ClickedAt > allEvents[j].ClickEventData.Payload.ClickedAt
	})

	filteredEvents := make([]model.ClickEvent, 0)
	for _, event := range allEvents {
		clickedAt := event.ClickEventData.Payload.ClickedAt
		if clickedAt >= startTimestampMs && clickedAt <= endTimestampMs {
			filteredEvents = append(filteredEvents, event)
		}
	}

	if limit > 0 && len(filteredEvents) > int(limit) {
		filteredEvents = filteredEvents[:int(limit)]
	}

	return filteredEvents, nil
}


// buildTableToFieldsMapping creates a mapping of table names to week field names
// based on the timestamp range. It handles both normal ranges and wrap-around scenarios.
func (c *ClickRetrieveHandler) buildTableToFieldsMapping(startTimestampMs int64, endTimestampMs int64, getTableNameFunc func(int) string) map[string][]string {
	lowerBound := utils.WeekFromTimestampMs(startTimestampMs) % 24
	upperBound := utils.WeekFromTimestampMs(endTimestampMs) % 24
	tablesToFields := make(map[string][]string)

	if lowerBound <= upperBound && utils.TimestampDiffInWeeks(startTimestampMs, endTimestampMs) <= 24 {
		// Normal case: range doesn't wrap around
		for i := lowerBound; i <= upperBound; i++ {
			bucketIdx := i / 8
			tableName := getTableNameFunc(bucketIdx)
			tablesToFields[tableName] = append(tablesToFields[tableName], "week_"+strconv.Itoa(i))
		}
	} else {
		// Wrap-around case: range spans across week boundary
		for i := lowerBound; i < 24; i++ {
			bucketIdx := i / 8
			tableName := getTableNameFunc(bucketIdx)
			tablesToFields[tableName] = append(tablesToFields[tableName], "week_"+strconv.Itoa(i))
		}
		for i := 0; i <= upperBound; i++ {
			bucketIdx := i / 8
			tableName := getTableNameFunc(bucketIdx)
			tablesToFields[tableName] = append(tablesToFields[tableName], "week_"+strconv.Itoa(i))
		}
	}

	return tablesToFields
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

func (c *ClickRetrieveHandler) fetchDataInParallel(tablesToFields map[string][]string, userId string) (map[string][]byte, error) {
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
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("panic occurred while fetching click data from table %s: %v", tblName, r)
					resultChan <- fetchResult{
						tableName: tblName,
						columns:   cols,
						err:       fmt.Errorf("panic during data fetch from table %s: %v", tblName, r),
					}
				}
				wg.Done()
			}()
			data, err := c.scyllaDb.RetrieveInteractions(tblName, userId, cols)
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
				// Field doesn't exist - use nil data
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

func (c *ClickRetrieveHandler) buildDeserialisedPermanentStorageDataBlocks(weekToData map[string][]byte) (map[string]*blocks.DeserializedPSDB, error) {
	weekToDeserializedBlocks := make(map[string]*blocks.DeserializedPSDB)
	for column, data := range weekToData {
		if len(data) == 0 {
			continue
		}
		ddb, err := blocks.DeserializePSDB(data, enum.InteractionTypeClick)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize data for column %s, error: %w", column, err)
		}
		weekToDeserializedBlocks[column] = ddb
	}
	return weekToDeserializedBlocks, nil
}
