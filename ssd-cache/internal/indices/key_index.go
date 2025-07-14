package indices


const (
	INIT_QUEUE_SIZE = 1000000
	INIT_MAP_SIZE = 1000000
)

type Meta struct {
	relativeOffset int32
	length         int32
	memtableId     int64
}

type KeyIndex struct {
	idx                   map[string]Meta
	queue                 []string
	memtableIdStartToQIdx map[int64]int
	activeMemtableId      int64
}

func NewKeyIndex() *KeyIndex {
	return &KeyIndex{
		idx:                   make(map[string]Meta),
		memtableIdStartToQIdx: make(map[int64]int),
		activeMemtableId:      0,
	}
}

func (ki *KeyIndex) Add(key string, meta Meta) {
	if ki.activeMemtableId != meta.memtableId {
		ki.activeMemtableId = meta.memtableId
		ki.memtableIdStartToQIdx[meta.memtableId] = len(ki.queue)
	}
	ki.idx[key] = meta
	ki.queue = append(ki.queue, key)
}

func (ki *KeyIndex) Get(key string) (Meta, bool) {
	meta, ok := ki.idx[key]