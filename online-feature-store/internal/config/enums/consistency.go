package enums

type Consistency string

const (
	ConsistencyUnknown  Consistency = "ConsistencyUnknown"
	ConsistencyStrong   Consistency = "ConsistencyStrong"
	ConsistencyEventual Consistency = "ConsistencyEventual"
)
