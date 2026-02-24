package retrieve

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateTimeRange_EndBeforeStart(t *testing.T) {
	start := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC).UnixMilli()

	err := validateTimeRange(start, end, 1)

	assert.ErrorIs(t, err, ErrInvalidTimeRange)
}

func TestValidateTimeRange_EndEqualsStart(t *testing.T) {
	// Past timestamp so we don't hit ErrEndTimeInFuture
	ts := time.Now().Add(-48 * time.Hour).UnixMilli()

	err := validateTimeRange(ts, ts, 1)

	assert.NoError(t, err)
}

func TestValidateTimeRange_EndInFuture(t *testing.T) {
	start := time.Now().Add(-1 * time.Hour).UnixMilli()
	end := time.Now().Add(1 * time.Hour).UnixMilli()

	err := validateTimeRange(start, end, 1)

	assert.ErrorIs(t, err, ErrEndTimeInFuture)
}

func TestValidateTimeRange_ExceedsMaxWeeks(t *testing.T) {
	// Use past timestamps so we don't hit ErrEndTimeInFuture
	// 25 weeks apart
	start := time.Now().Add(-26 * 7 * 24 * time.Hour).UnixMilli()
	end := time.Now().Add(-1 * 7 * 24 * time.Hour).UnixMilli()

	err := validateTimeRange(start, end, 1)

	assert.ErrorIs(t, err, ErrTimeRangeExceeded)
}

func TestValidateTimeRange_Valid(t *testing.T) {
	start := time.Now().Add(-2 * 24 * time.Hour).UnixMilli()
	end := time.Now().Add(-1 * time.Hour).UnixMilli()

	err := validateTimeRange(start, end, 1)

	assert.NoError(t, err)
}

func TestValidateTimeRange_InvalidLimit(t *testing.T) {
	start := time.Now().Add(-48 * time.Hour).UnixMilli()
	end := time.Now().Add(-24 * time.Hour).UnixMilli()

	errZero := validateTimeRange(start, end, 0)
	assert.ErrorIs(t, errZero, ErrInvalidLimit)

	errNeg := validateTimeRange(start, end, -1)
	assert.ErrorIs(t, errNeg, ErrInvalidLimit)
}

func TestValidateTimeRange_ValidExactly24Weeks(t *testing.T) {
	// Exactly 24 weeks apart, both in the past
	start := time.Now().Add(-25 * 7 * 24 * time.Hour).UnixMilli()
	end := time.Now().Add(-1 * 7 * 24 * time.Hour).UnixMilli()

	err := validateTimeRange(start, end, 1)

	assert.NoError(t, err)
}
