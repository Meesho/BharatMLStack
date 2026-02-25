package constants

// Week-column and rolling window constants for interaction-store (Scylla schema: week_0..week_23).
const (
	// WeekColumnPrefix is the prefix for week column names in Scylla (e.g. week_0, week_1).
	WeekColumnPrefix = "week_"

	// TotalWeeks is the number of weeks in the rolling window (week_0..week_23).
	TotalWeeks = 24

	// MaxWeekIndex is the maximum week index (TotalWeeks - 1).
	MaxWeekIndex = 23

	// WeeksPerBucket is the number of weeks per Scylla bucket (bucket0: 0-7, bucket1: 8-15, bucket2: 16-23).
	WeeksPerBucket = 8

	// MaxRetrieveLimit is the maximum number of events returned per retrieve request.
	MaxRetrieveLimit = 2000

	// MaxOrderEventsPerWeek is the maximum order events retained per week column.
	MaxOrderEventsPerWeek = 500

	// MaxClickEventsPerWeek is the maximum click events retained per week column.
	MaxClickEventsPerWeek = 500
)
