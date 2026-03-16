package retrieve

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/constants"
	blocks "github.com/Meesho/BharatMLStack/interaction-store/internal/data/block"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/enum"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/model"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/scylla"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
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
	if err := validateTimeRange(startTimestampMs, endTimestampMs, limit); err != nil {
		return nil, err
	}

	limit = capLimit(limit, constants.MaxRetrieveLimit)

	orderedWeeks := c.buildOrderedWeeks(startTimestampMs, endTimestampMs)
	tablesToFields := c.buildTableToFieldsMapping(orderedWeeks)

	weekToData, err := c.fetchDataInParallel(tablesToFields, userId)
	if err != nil {
		return nil, err
	}

	weekToEvents, err := c.deserializeWeeks(weekToData, userId)
	if err != nil {
		return nil, err
	}

	return c.mergeFilterAndLimit(orderedWeeks, weekToEvents, startTimestampMs, endTimestampMs, int(limit)), nil
}

// buildOrderedWeeks returns week indices in descending temporal order,
// handling wrap-around (e.g. weeks 21,22,23,0 → descending: 0, 23, 22, 21).
func (c *ClickRetrieveHandler) buildOrderedWeeks(startTimestampMs int64, endTimestampMs int64) []int {
	lowerBound := utils.WeekFromTimestampMs(startTimestampMs) % constants.TotalWeeks
	upperBound := utils.WeekFromTimestampMs(endTimestampMs) % constants.TotalWeeks
	diffWeeks := utils.TimestampDiffInWeeks(startTimestampMs, endTimestampMs)

	var weeks []int
	if lowerBound <= upperBound && diffWeeks <= constants.TotalWeeks {
		if lowerBound == upperBound && diffWeeks == constants.TotalWeeks {
			// Full cycle: start/end map to same week index but range spans TotalWeeks
			weeks = make([]int, 0, constants.TotalWeeks)
			for i := constants.MaxWeekIndex; i >= 0; i-- {
				weeks = append(weeks, i)
			}
		} else {
			weeks = make([]int, 0, upperBound-lowerBound+1)
			for i := upperBound; i >= lowerBound; i-- {
				weeks = append(weeks, i)
			}
		}
	} else {
		capacity := (constants.TotalWeeks - lowerBound) + upperBound + 1
		weeks = make([]int, 0, capacity)
		for i := upperBound; i >= 0; i-- {
			weeks = append(weeks, i)
		}
		for i := constants.MaxWeekIndex; i >= lowerBound; i-- {
			weeks = append(weeks, i)
		}
	}
	return weeks
}

func (c *ClickRetrieveHandler) buildTableToFieldsMapping(orderedWeeks []int) map[string][]string {
	tablesToFields := make(map[string][]string)
	for _, week := range orderedWeeks {
		tableName := getTableName(week / constants.WeeksPerBucket)
		tablesToFields[tableName] = append(tablesToFields[tableName], constants.WeekColumnPrefix+strconv.Itoa(week))
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
	totalColumns := 0
	for _, fields := range tablesToFields {
		totalColumns += len(fields)
	}
	weekToData := make(map[string][]byte, totalColumns)
	var mu sync.Mutex
	g := new(errgroup.Group)

	for tableName, fields := range tablesToFields {
		if len(fields) == 0 {
			continue
		}
		tblName, cols := tableName, fields
		g.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msgf("panic occurred while fetching click data from table %s: %v", tblName, r)
					err = fmt.Errorf("panic during data fetch from table %s: %v", tblName, r)
				}
			}()
			data, fetchErr := c.scyllaDb.RetrieveInteractions(tblName, userId, cols)
			if fetchErr != nil {
				return fmt.Errorf("failed to retrieve data for table: %s, error: %w", tblName, fetchErr)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, column := range cols {
				value, ok := data[column]
				if !ok {
					weekToData[column] = nil
					continue
				}
				rawData, ok := value.([]byte)
				if !ok {
					return fmt.Errorf("unexpected data type for column %s in table %s: got %T, want []byte", column, tblName, value)
				}
				weekToData[column] = rawData
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return weekToData, nil
}

// deserializeWeeks deserializes per-week data. When more than one week has data,
// work is done in parallel to reduce latency (deserialize time ≈ max over weeks
// instead of sum). For a single week the sequential path avoids goroutine/channel
// overhead.
func (c *ClickRetrieveHandler) deserializeWeeks(weekToData map[string][]byte, userId string) (map[string][]model.ClickEvent, error) {
	weekToEvents := make(map[string][]model.ClickEvent, len(weekToData))

	// Collect (column, data) pairs with non-empty data for deterministic iteration.
	var work []struct {
		column string
		data   []byte
	}
	for column, data := range weekToData {
		if len(data) == 0 {
			continue
		}
		work = append(work, struct {
			column string
			data   []byte
		}{column, data})
	}

	if len(work) == 0 {
		return weekToEvents, nil
	}
	if len(work) == 1 {
		w := work[0]
		ddb, err := blocks.DeserializePSDB(w.data, enum.InteractionTypeClick)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize data for column %s: %w", w.column, err)
		}
		eventsRaw, err := ddb.RetrieveEventData(userId)
		if err != nil {
			return nil, err
		}
		events, ok := eventsRaw.([]model.ClickEvent)
		if !ok {
			return nil, fmt.Errorf("unexpected event data type for column %s: got %T", w.column, eventsRaw)
		}
		weekToEvents[w.column] = events
		return weekToEvents, nil
	}

	g := new(errgroup.Group)
	var mu sync.Mutex
	for _, w := range work {
		w := w
		g.Go(func() error {
			ddb, err := blocks.DeserializePSDB(w.data, enum.InteractionTypeClick)
			if err != nil {
				return fmt.Errorf("failed to deserialize data for column %s: %w", w.column, err)
			}
			eventsRaw, err := ddb.RetrieveEventData(userId)
			if err != nil {
				return err
			}
			events, ok := eventsRaw.([]model.ClickEvent)
			if !ok {
				return fmt.Errorf("unexpected event data type for column %s: got %T", w.column, eventsRaw)
			}
			mu.Lock()
			weekToEvents[w.column] = events
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return weekToEvents, nil
}

// mergeFilterAndLimit concatenates pre-sorted (descending) per-week events,
// applies time-range filter, and stops at limit.
func (c *ClickRetrieveHandler) mergeFilterAndLimit(orderedWeeks []int, weekToEvents map[string][]model.ClickEvent, startTimestampMs int64, endTimestampMs int64, limit int) []model.ClickEvent {
	result := make([]model.ClickEvent, 0, limit)

	for _, week := range orderedWeeks {
		events, ok := weekToEvents[constants.WeekColumnPrefix+strconv.Itoa(week)]
		if !ok || len(events) == 0 {
			continue
		}
		for i := range events {
			ts := events[i].ClickEventData.Payload.ClickedAt
			if ts > endTimestampMs {
				continue
			}
			if ts < startTimestampMs {
				break
			}
			result = append(result, events[i])
			if limit > 0 && len(result) >= limit {
				return result
			}
		}
	}

	return result
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
