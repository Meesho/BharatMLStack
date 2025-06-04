package handler

import "sync"

var (
	once   sync.Once
	config Config
)

func NewConfigHandler(version int) Config {
	switch version {
	case 1:
		return InitV1ConfigHandler()
	default:
		return nil
	}
}

func ResetConfigForTests() {
	once = sync.Once{} // Re-instantiate the sync.Once
	config = nil       // Reset the handler instance
}
