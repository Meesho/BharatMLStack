package network

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/network/client"
	"github.com/rs/zerolog/log"
)

const (
	RETRY_AFTER_N_REQUESTS = 10
)

// Message types for the unified channel
type Message any

type RequestMessage struct {
	Key          string
	ResponseChan chan ResponseMessage
	IP           string
}

type ResponseMessage struct {
	Key  string
	Data []byte
}

type CancelMessage struct {
	Key          string
	ResponseChan chan ResponseMessage
}

type ClientManager struct {
	messageChannel   chan Message
	responseChannels map[string][]chan ResponseMessage
	client           client.Client
}

func NewClientManager(serverPort int) *ClientManager {
	// TODO: Tune the buffer sizes
	messageChannel := make(chan Message, 10000)
	clientResponseChannel := make(chan []byte, 10000)

	client := &ClientManager{
		messageChannel:   messageChannel,
		responseChannels: make(map[string][]chan ResponseMessage),
		client:           client.NewUDPClient(MAX_PACKET_SIZE_IN_BYTES, serverPort, clientResponseChannel),
	}
	go client.start()
	go client.handleResponses(clientResponseChannel)

	return client
}

func (c *ClientManager) start() {
	for message := range c.messageChannel {
		switch msg := message.(type) {
		case RequestMessage:
			c.handleRequest(msg)
		case ResponseMessage:
			c.handleResponse(msg)
		case CancelMessage:
			c.handleCancel(msg)
		}
	}
}

func (c *ClientManager) handleRequest(msg RequestMessage) {
	metric.Count("p2p.client.manager.requests.total", 1, []string{})
	if _, ok := c.responseChannels[msg.Key]; !ok {
		c.responseChannels[msg.Key] = make([]chan ResponseMessage, 0)
	}

	// Send the request to the server after RETRY_AFTER_N_REQUESTS requests to account for any dropped packets
	if (len(c.responseChannels[msg.Key]) % RETRY_AFTER_N_REQUESTS) == 0 {
		go c.client.SendMessage([]byte(msg.Key), msg.IP)
	}
	c.responseChannels[msg.Key] = append(c.responseChannels[msg.Key], msg.ResponseChan)
}

func (c *ClientManager) handleResponse(msg ResponseMessage) {
	responseChannelsSize := len(c.responseChannels[msg.Key])
	if responseChannelsSize == 0 {
		return
	}

	metric.Count("p2p.client.manager.responses.total", int64(responseChannelsSize), []string{})
	for _, responseChan := range c.responseChannels[msg.Key] {
		select {
		case responseChan <- msg:
		default:
			// Channel is full or closed, skip
		}
	}
	delete(c.responseChannels, msg.Key)
}

func (c *ClientManager) handleCancel(msg CancelMessage) {
	responseChannels, ok := c.responseChannels[msg.Key]
	if !ok {
		return
	}
	for i, responseChan := range responseChannels {
		if responseChan == msg.ResponseChan {
			c.responseChannels[msg.Key] = append(c.responseChannels[msg.Key][:i], c.responseChannels[msg.Key][i+1:]...)
			break
		}
	}
	close(msg.ResponseChan)
	if len(c.responseChannels[msg.Key]) == 0 {
		delete(c.responseChannels, msg.Key)
	}
}

// handleResponses reads from clientResponseChannel and converts to ResponseMessage
func (c *ClientManager) handleResponses(clientResponseChannel <-chan []byte) {
	for response := range clientResponseChannel {
		keyIndex := bytes.IndexByte(response, RESPONSE_PACKET_KEY_VALUE_SEPARATOR)
		if keyIndex == -1 {
			// Drop invalid response packets
			metric.Count("p2p.cache.store.values.error", 1, []string{"reason", "invalid_response_packet"})
			continue
		}
		data := response[keyIndex+1:]
		if len(data) == 1 && data[0] == VALUE_NOT_FOUND_RESPONSE {
			data = nil
		}
		select {
		case c.messageChannel <- ResponseMessage{
			Key:  string(response[:keyIndex]),
			Data: data,
		}:
		default:
			metric.Count("p2p.cache.store.values.error", 1, []string{"reason", "response_channel_full"})
		}
	}
}

// GetData sends a request and returns a channel for the response
func (c *ClientManager) GetData(key string, ip string) *ResponseMessage {
	responseChan := make(chan ResponseMessage, 1)
	select {
	case c.messageChannel <- RequestMessage{
		Key:          key,
		ResponseChan: responseChan,
		IP:           ip,
	}:
	default:
		metric.Count("p2p.cache.store.keys.dropped", 1, []string{"sendTo", ip, "reason", "request_channel_full"})
		log.Error().Msgf("GetData: request channel is full for key %s from pod %s", key, ip)
		return nil
	}

	select {
	case value := <-responseChan:
		return &value
	case <-time.After(REQUEST_TIMEOUT):
		log.Error().Msgf("GetData: request timed out for key %s from pod %s", key, ip)
		metric.Count("p2p.cache.store.keys.dropped", 1, []string{"sendTo", ip, "reason", "timeout"})
		go c.CancelRequest(key, responseChan)
		return nil
	}
}

func (c *ClientManager) SetData(key string, value []byte, ttlInSeconds int, ip string) {

	// follows the message structure of [0 <key> 0 <ttl in secs for 8 bytes> <value>]
	// Given that the key is a string, it will never have a zero byte in it. So, safe to use it as a delimiter.
	message := []byte{}
	message = append(message, SET_DATA_PACKET_START_BYTE_IDENTIFIER)
	message = append(message, key...)

	message = append(message, SET_DATA_PACKET_KEY_TTL_SEPARATOR)
	ttlBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(ttlBytes, uint64(ttlInSeconds))
	message = append(message, ttlBytes...)

	message = append(message, value...)

	log.Debug().Msgf("Sending key %s with ttl %d in seconds with value %v, final message: %v", key, ttlInSeconds, value, message)
	go c.client.SendMessage(message, ip)
}

func (c *ClientManager) CancelRequest(key string, responseChan chan ResponseMessage) {
	c.messageChannel <- CancelMessage{
		Key:          key,
		ResponseChan: responseChan,
	}
}
