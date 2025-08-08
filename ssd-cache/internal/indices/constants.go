package indices

const (
	LENGTH_MASK      = (1 << 16) - 1
	LAST_ACCESS_MASK = (1 << 24) - 1
	FREQ_MASK        = (1 << 24) - 1
	H10_MASK         = (1 << 10) - 1
	EXPTIME_MASK     = (1 << 54) - 1
)
