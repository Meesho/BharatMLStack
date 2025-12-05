package handler

var (
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
