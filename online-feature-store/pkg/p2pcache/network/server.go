package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"runtime"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/storage"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog/log"
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

	if buf[0] == SET_DATA_PACKET_START_BYTE_IDENTIFIER {
		// create copy of buf as it is reused for next message
		bufCopy := make([]byte, len(buf))
		copy(bufCopy, buf)
		go s.handleSetDataPacket(bufCopy)
		return gnet.None
	} else {
		return s.handleGetDataPacket(c, buf)
	}
}

func (s *Server) handleSetDataPacket(buf []byte) {
	// message structure: [0 <key> 0 <ttl in secs for 8 bytes> <value>]
	keyIndex := bytes.IndexByte(buf[1:], SET_DATA_PACKET_KEY_TTL_SEPARATOR) + 1
	if keyIndex == -1 {
		// drop invalid packets
		metric.Count("p2p.cache.server.keys.error", 1, []string{"reason", "invalid_set_data_packet"})
		return
	}

	key := string(buf[1:keyIndex])
	ttl := binary.BigEndian.Uint64(buf[keyIndex+1 : keyIndex+9])
	value := buf[keyIndex+9:]

	log.Debug().Msgf("Received key %s with ttl %d in seconds with value %v, buf: %v", key, ttl, value, buf)
	s.cacheStore.SetIntoOwnPartitionCache(key, value, int(ttl))
}

func (s *Server) handleGetDataPacket(c gnet.Conn, buf []byte) gnet.Action {
	response := append([]byte{}, buf...)
	response = append(response, RESPONSE_PACKET_KEY_VALUE_SEPARATOR)

	key := string(buf)
	value, err := s.cacheStore.Get(key)
	log.Debug().Msgf("Value for key %s from server cache: %v", key, value)

	// Let the client know if the value is not found or if the response is too large
	if err != nil {
		metric.Count("p2p.cache.server.keys", 1, []string{"type", "miss"})
		response = append(response, VALUE_NOT_FOUND_RESPONSE)
	} else if len(value)+len(response) > MAX_PACKET_SIZE_IN_BYTES {
		metric.Count("p2p.cache.server.keys", 1, []string{"type", "too_large"})
	} else {
		metric.Count("p2p.cache.server.keys", 1, []string{"type", "hit"})
		// TODO: Compress the value sent over network
		response = append(response, value...)
	}

	c.Write(response)
	return gnet.None
}

func (s *Server) start(port int) {
	// TODO: Tune buffer sizes
	err := gnet.Run(s, fmt.Sprintf("udp://:%d", port),
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
		gnet.WithLockOSThread(false),
		gnet.WithReadBufferCap(1024*1024*100),
		gnet.WithWriteBufferCap(1024*1024*1000),
		gnet.WithNumEventLoop(runtime.NumCPU()))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start P2P cache server")
		panic(err)
	}
}
