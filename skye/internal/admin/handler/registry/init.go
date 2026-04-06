package registry

import "sync"

var (
	DefaultVersion = 1
)

func NewHandler(version int) Manager {
	switch version {
	case DefaultVersion:
		return initRegistryHandler()
	default:
		return nil
	}
}

func ResetConfigForTests() {
	once = sync.Once{} // Re-instantiate the sync.Once
	manager = nil      // Reset the handler instance
}
