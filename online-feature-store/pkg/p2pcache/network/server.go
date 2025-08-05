package network

import (
	"fmt"
	"runtime"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/storage"
	"github.com/panjf2000/gnet/v2"
)

func NewServer(port int, cacheStore *storage.CacheStore) *Server {
	server := &Server{
		cacheStore: cacheStore,
	}
	go server.start(port)
	return server
}

type Server struct {
	gnet.BuiltinEventEngine

	cacheStore *storage.CacheStore
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	buf, err := c.Next(-1)
	if err != nil {
		return gnet.Close
	}

	response := append([]byte{}, buf...)
	response = append(response, RESPONSE_PACKET_KEY_VALUE_SEPARATOR)

	value, err := s.cacheStore.Get(string(buf))

	// Let the client know if the value is not found or if the response is too large
	if err != nil || len(value)+len(response) > MAX_PACKET_SIZE_IN_BYTES {
		response = append(response, VALUE_NOT_FOUND_RESPONSE)
	} else {
		response = append(response, value...)
	}

	c.Write(response)
	return gnet.None
}

func (s *Server) start(port int) {
	// TODO: Tune buffer sizes
	gnet.Run(s, fmt.Sprintf("udp://:%d", port),
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithLockOSThread(false),
		gnet.WithReadBufferCap(1024*1024*100),
		gnet.WithWriteBufferCap(1024*1024*1000),
		gnet.WithNumEventLoop(runtime.NumCPU()))
}
