package aggregator

import "sync"

var (
	once sync.Once
)

type Handler interface {
	Process(payload Payload) (*Response, error)
}
