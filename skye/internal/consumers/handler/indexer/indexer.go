package indexer

import "sync"

var (
	once sync.Once
)

type Handler interface {
	Process(event Event) error
}
