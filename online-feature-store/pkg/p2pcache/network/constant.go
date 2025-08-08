package network

import "time"

const (
	RESPONSE_PACKET_KEY_VALUE_SEPARATOR = byte(0)
	VALUE_NOT_FOUND_RESPONSE            = byte(0)

	MAX_PACKET_SIZE_IN_BYTES = 1024 * 4 // 4KB

	REQUEST_TIMEOUT = 10 * time.Millisecond
)
