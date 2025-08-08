//go:build darwin

package client

import (
	"net"

	"github.com/rs/zerolog/log"
)

type DarwinUDPClient struct {
	outputChannel chan []byte
	conn          *net.UDPConn
	serverPort    int
}

func NewUDPClient(maxPacketSizeInBytes int, serverPort int, outputChannel chan []byte) Client {
	udpClient := &DarwinUDPClient{
		outputChannel: outputChannel,
		serverPort:    serverPort,
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

func (c *DarwinUDPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
