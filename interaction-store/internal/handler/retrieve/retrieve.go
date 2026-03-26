package retrieve

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/interaction-store/internal/constants"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/utils"
)

var (
	ErrTimeRangeExceeded = errors.New("startTimestampMs and endTimestampMs cannot be more than 24 weeks apart")
	ErrEndTimeInFuture   = errors.New("endTimestampMs cannot be in the future")
	ErrInvalidTimeRange  = errors.New("endTimestampMs must be greater than or equal to startTimestampMs")
	ErrInvalidLimit      = errors.New("limit must be greater than zero")
)

type RetrieveHandler[T any] interface {
	Retrieve(userId string, startTimestampMs int64, endTimestampMs int64, limit int32) (data T, err error)
}

func validateTimeRange(startTimestampMs, endTimestampMs int64, limit int32) error {
	if endTimestampMs < startTimestampMs {
		return ErrInvalidTimeRange
	}
	if endTimestampMs > time.Now().UnixMilli() {
		return ErrEndTimeInFuture
	}
	if utils.TimestampDiffInWeeks(startTimestampMs, endTimestampMs) > constants.TotalWeeks {
		return ErrTimeRangeExceeded
	}
	if limit <= 0 {
		return ErrInvalidLimit
	}
	return nil
}

func capLimit(limit int32, maxLimit int32) int32 {
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
