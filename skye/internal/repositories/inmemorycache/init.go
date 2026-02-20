package inmemorycache

var (
	DefaultVersion = 1
)

func NewRepository(version int) Database {
	switch version {
	case 1:
		return initFreeCache()
	default:
		return nil
	}
}
