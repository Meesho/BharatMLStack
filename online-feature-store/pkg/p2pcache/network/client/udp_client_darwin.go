//go:build darwin

package client

import (
	"fmt"
	"net"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
)

type DarwinUDPClient struct {
	outputChannel        chan []byte
	conn                 *net.UDPConn
	serverPort           int
	maxPacketSizeInBytes int
}

func NewUDPClient(maxPacketSizeInBytes int, serverPort int, outputChannel chan []byte) Client {
	udpClient := &DarwinUDPClient{
		outputChannel:        outputChannel,
		serverPort:           serverPort,
		maxPacketSizeInBytes: maxPacketSizeInBytes,
	}

	// Create UDP connection bound to any available port
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	udpClient.conn = conn

	go udpClient.startReceiver(maxPacketSizeInBytes)
	return udpClient
}

func (c *DarwinUDPClient) startReceiver(maxPacketSizeInBytes int) {
	buf := make([]byte, maxPacketSizeInBytes)

	for {
		n, _, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			log.Error().Msgf("Error reading UDP message: %v", err)
			continue
		}

		// Send received message to output channel
		c.outputChannel <- buf[:n]
	}
}

func (c *DarwinUDPClient) SendMessage(message []byte, ip string) error {
	if len(message) > c.maxPacketSizeInBytes {
		metric.Count("p2p.cache.store.values.error", 1, []string{"reason", "send_message_greater_than_max_packet_size"})
		return fmt.Errorf("message size is greater than max packet size: %d > %d", len(message), c.maxPacketSizeInBytes)
	}
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return net.InvalidAddrError("invalid IP address")
	}

	udpAddr := &net.UDPAddr{
		IP:   ipAddr,
		Port: c.serverPort,
	}

	_, err := c.conn.WriteToUDP(message, udpAddr)
	return err
}
