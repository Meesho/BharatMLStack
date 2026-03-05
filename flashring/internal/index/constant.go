package index

const (
	LENGTH_MASK        = (1 << 16) - 1
	DELTA_EXPTIME_MASK = (1 << 16) - 1
	LAST_ACCESS_MASK   = (1 << 16) - 1
	FREQ_MASK          = (1 << 16) - 1
	PREV_MASK          = (1 << 32) - 1
	NEXT_MASK          = (1 << 32) - 1

	MEM_ID_MASK = (1 << 32) - 1
	OFFSET_MASK = (1 << 32) - 1

	LENGTH_SHIFT        = 48
	DELTA_EXPTIME_SHIFT = 32
	LAST_ACCESS_SHIFT   = 16
	FREQ_SHIFT          = 0

	MEM_ID_SHIFT = 32
	OFFSET_SHIFT = 0
)
