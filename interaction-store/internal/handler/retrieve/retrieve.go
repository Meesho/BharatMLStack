package retrieve

import (
	"errors"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
)

const maxWeeksApart = 24

var ErrTimeRangeExceeded = errors.New("startTimestampMs and endTimestampMs cannot be more than 24 weeks apart")

type RetrieveHandler[T any] interface {
	Retrieve(userId string, startTimestampMs int64, endTimestampMs int64, limit int32) (data T, err error)
}

func validateTimeRange(startTimestampMs, endTimestampMs int64) error {
	if utils.TimestampDiffInWeeks(startTimestampMs, endTimestampMs) > maxWeeksApart {
		return ErrTimeRangeExceeded
	}
	return nil
}

func capLimit(limit int32, maxLimit int32) int32 {
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
