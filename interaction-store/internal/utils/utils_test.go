package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWeekFromTimestampMs(t *testing.T) {
	// Jan 15, 2024 is in ISO week 3
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	week := WeekFromTimestampMs(ts)
	assert.GreaterOrEqual(t, week, 1)
	assert.LessOrEqual(t, week, 53)
}

func TestWeekFromTimestampMs_ConsistentForSameWeek(t *testing.T) {
	// Same calendar day at different times should yield same week
	ts := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	tsLater := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).UnixMilli()
	assert.Equal(t, WeekFromTimestampMs(ts), WeekFromTimestampMs(tsLater))
}

func TestTimestampDiffInWeeks_SameTime(t *testing.T) {
	ts := time.Now().UnixMilli()
	assert.Equal(t, 0, TimestampDiffInWeeks(ts, ts))
}

func TestTimestampDiffInWeeks_OneWeek(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()
	assert.Equal(t, 1, TimestampDiffInWeeks(start, end))
	assert.Equal(t, 1, TimestampDiffInWeeks(end, start))
}

func TestTimestampDiffInWeeks_OrderIndependent(t *testing.T) {
	ts1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	ts2 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	assert.Equal(t, TimestampDiffInWeeks(ts1, ts2), TimestampDiffInWeeks(ts2, ts1))
}

func TestTimestampDiffInWeeks_LessThanOneWeek(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC).UnixMilli()
	assert.Equal(t, 0, TimestampDiffInWeeks(start, end))
}

func TestTimestampDiffInWeeks_25Weeks(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2024, 6, 24, 0, 0, 0, 0, time.UTC).UnixMilli() // ~25 weeks
	diff := TimestampDiffInWeeks(start, end)
	assert.GreaterOrEqual(t, diff, 24)
}
