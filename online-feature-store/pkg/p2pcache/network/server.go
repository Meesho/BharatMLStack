package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"runtime"
	"sync"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/storage"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog/log"
)

const (
	TOTAL_READ_BUFFER_CAP_BYTES  = 1024 * 1024 * 100
	TOTAL_WRITE_BUFFER_CAP_BYTES = 1024 * 1024 * 1000
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

var packetPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, MAX_PACKET_SIZE_IN_BYTES)
		return &b
	},
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	buf, err := c.Next(-1)
	if err != nil {
		return gnet.Close
	}

	if buf[0] == SET_DATA_PACKET_START_BYTE_IDENTIFIER {
		return s.handleSetDataPacket(buf)
	} else {
		return s.handleGetDataPacket(c, buf)
	}
}

func (s *Server) handleSetDataPacket(buf []byte) gnet.Action {
	// message structure: [0 <key> 0 <ttl in secs for 8 bytes> <value>]
	keyIndex := bytes.IndexByte(buf[1:], SET_DATA_PACKET_KEY_TTL_SEPARATOR) + 1
	if keyIndex == 0 {
		// drop invalid packets
		metric.Count("p2p.cache.server.keys.error", 1, []string{"reason", "invalid_set_data_packet"})
		return gnet.None
	}

	key := string(buf[1:keyIndex])
	ttl := binary.BigEndian.Uint64(buf[keyIndex+1 : keyIndex+9])
	value := buf[keyIndex+9:]

	log.Debug().Msgf("Received key %s with ttl %d in seconds with value %v, buf: %v", key, ttl, value, buf)
	s.cacheStore.SetIntoOwnPartitionCache(key, value, int(ttl))
	return gnet.None
}

func (s *Server) handleGetDataPacket(c gnet.Conn, buf []byte) gnet.Action {
	responseBuf := *(packetPool.Get().(*[]byte))
	defer packetPool.Put(&responseBuf)

	n := copy(responseBuf, buf)
	responseBuf[n] = RESPONSE_PACKET_KEY_VALUE_SEPARATOR
	n++

	key := string(buf)
	value, err := s.cacheStore.Get(key)
	log.Debug().Msgf("Value for key %s from server cache: %v", key, value)

	// Let the client know if the value is not found or if the response is too large
	if err != nil {
		metric.Count("p2p.cache.server.keys", 1, []string{"reason", "miss"})
		responseBuf[n] = VALUE_NOT_FOUND_RESPONSE
		n++
	} else if len(value)+n > MAX_PACKET_SIZE_IN_BYTES {
		metric.Count("p2p.cache.server.keys", 1, []string{"reason", "too_large"})
		responseBuf[n] = VALUE_NOT_FOUND_RESPONSE
		n++
	} else {
		metric.Count("p2p.cache.server.keys", 1, []string{"reason", "hit"})
		// TODO: Compress the value sent over network
		n += copy(responseBuf[n:], value)
	}

	c.Write(responseBuf[:n])
	return gnet.None
}

func (s *Server) start(port int) {
	// TODO: Tune buffer sizes
	err := gnet.Run(s, fmt.Sprintf("udp://:%d", port),
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithLockOSThread(false),
		gnet.WithReadBufferCap(TOTAL_READ_BUFFER_CAP_BYTES/runtime.NumCPU()),
		gnet.WithWriteBufferCap(TOTAL_WRITE_BUFFER_CAP_BYTES/runtime.NumCPU()),
		gnet.WithNumEventLoop(runtime.NumCPU()))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start P2P cache server")
		panic(err)
	}
}
