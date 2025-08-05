package network

import (
	"bytes"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/p2pcache/network/client"
)

const (
	RETRY_AFTER_N_REQUESTS = 10
)

// Message types for the unified channel
type Message interface{}

type RequestMessage struct {
	Key          string
	ResponseChan chan ResponseMessage
	IP           string
}

type ResponseMessage struct {
	Key  string
	Data []byte
}

type ClientManager struct {
	messageChannel   chan Message
	responseChannels map[string][]chan ResponseMessage
	client           client.Client
}

func NewClientManager(serverPort int) *ClientManager {
	// TODO: Tune the buffer sizes
	messageChannel := make(chan Message, 1000)
	clientResponseChannel := make(chan []byte, 1000)

	client := &ClientManager{
		messageChannel:   messageChannel,
		responseChannels: make(map[string][]chan ResponseMessage),
		client:           client.NewUDPClient(MAX_PACKET_SIZE_IN_BYTES, serverPort, clientResponseChannel),
	}
	go client.start()
	go client.handleResponses(clientResponseChannel)

	// TODO: Handle cleaning up responseChannels which are orphaned
	// Can happen due to client timing out and packets dropping
	return client
}

func (c *ClientManager) start() {
	for message := range c.messageChannel {
		switch msg := message.(type) {
		case RequestMessage:
			c.handleRequest(msg)
		case ResponseMessage:
			c.handleResponse(msg)
		}
	}
}

func (c *ClientManager) handleRequest(msg RequestMessage) {
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
	responseChannels, ok := c.responseChannels[msg.Key]
	if !ok {
		return
	}

	for _, responseChan := range responseChannels {
		select {
		case responseChan <- msg:
		default:
			// Channel is full or closed, skip
		}
	}
	delete(c.responseChannels, msg.Key)
}

// handleResponses reads from clientResponseChannel and converts to ResponseMessage
func (c *ClientManager) handleResponses(clientResponseChannel <-chan []byte) {
	for response := range clientResponseChannel {
		keyIndex := bytes.IndexByte(response, RESPONSE_PACKET_KEY_VALUE_SEPARATOR)
		if keyIndex == -1 {
			// Drop invalid response packets
			continue
		}
		data := response[keyIndex+1:]
		if len(data) == 1 && data[0] == VALUE_NOT_FOUND_RESPONSE {
			data = nil
		}
		c.messageChannel <- ResponseMessage{
			Key:  string(response[:keyIndex]),
			Data: data,
		}
	}
}

// GetData sends a request and returns a channel for the response
func (c *ClientManager) GetData(key string, ip string) chan ResponseMessage {
	responseChan := make(chan ResponseMessage, 1)
	c.messageChannel <- RequestMessage{
		Key:          key,
		ResponseChan: responseChan,
		IP:           ip,
	}
	return responseChan
}
