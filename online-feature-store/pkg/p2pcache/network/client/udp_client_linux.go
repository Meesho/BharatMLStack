//go:build linux

package client

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

const (
	EVENTS_BUFFER_SIZE = 1024
)

type LinuxUDPClient struct {
	outputChannel        chan []byte
	fd                   int
	epfd                 int
	serverPort           int
	maxPacketSizeInBytes int
	addrCache            *ds.SyncMapWithTtl[string, *unix.SockaddrInet4]
}

const (
	addrTTL = 5 * time.Hour
)

func NewUDPClient(maxPacketSizeInBytes int, serverPort int, outputChannel chan []byte) Client {
	udpClient := &LinuxUDPClient{
		outputChannel:        outputChannel,
		serverPort:           serverPort,
		maxPacketSizeInBytes: maxPacketSizeInBytes,
	}
	fd, err := udpClient.createSocket()
	if err != nil {
		panic(err)
	}
	udpClient.fd = fd

	epfd, err := udpClient.createEpoll(fd)
	if err != nil {
		panic(err)
	}
	udpClient.epfd = epfd

	udpClient.addrCache = ds.NewSyncMapWithTtl[string, *unix.SockaddrInet4]()

	go udpClient.startReceiver()
	return udpClient
}

func (c *LinuxUDPClient) createSocket() (int, error) {
	// Create a socket for UDP communication
	// AF_INET: IPv4 address family
	// SOCK_DGRAM: Datagram socket for UDP protocol
	// SOCK_NONBLOCK: Non-blocking socket. Operations return immediately rather than waiting for packets. Returns error if operation cannot be completed immediately.
	// IPPROTO_UDP: UDP protocol
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_NONBLOCK, unix.IPPROTO_UDP)
	if err != nil {
		return 0, err
	}

	// TODO: Tune socket receive and send buffer sizes
	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, 64*1024*1024)
	if err != nil {
		return 0, err
	}
	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, 64*1024*1024)
	if err != nil {
		return 0, err
	}

	// Bind the socket to a port. 0 means any available port.
	err = unix.Bind(fd, &unix.SockaddrInet4{Port: 0})
	if err != nil {
		return 0, err
	}
	return fd, nil
}

func (c *LinuxUDPClient) createEpoll(fd int) (int, error) {
	// Epoll is used to monitor the socket for incoming data.
	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		return 0, err
	}

	// Add the socket to the epoll instance.
	// EPOLL_CTL_ADD: Add a file descriptor to the epoll instance.
	// Fd: File descriptor of the socket to monitor.
	// Ev: Event to monitor. EPOLLIN: Event for data to read.
	ev := &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}
	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, ev)
	if err != nil {
		return 0, err
	}
	return epfd, nil
}

func (c *LinuxUDPClient) startReceiver() {
	// Buffer to hold incoming single packet of data
	buf := make([]byte, c.maxPacketSizeInBytes)

	// Buffer to hold epoll events. EpollWait returns up to EVENTS_BUFFER_SIZE events per call.
	// If more than EVENTS_BUFFER_SIZE file descriptors are ready, remaining events are returned on subsequent EpollWait calls.
	events := make([]unix.EpollEvent, EVENTS_BUFFER_SIZE)

	for {
		// Wait for epoll event until an event is ready indefinitely(-1)
		numEvents, err := unix.EpollWait(c.epfd, events, -1)
		if numEvents == EVENTS_BUFFER_SIZE {
			metric.Count("p2p.cache.client.event_buffer_full", 1, []string{})
		}

		// EINTR: Interrupted system call are common in Linux due to user defined signals like log-rotation. We avoid logging expected errors.
		if err == unix.EINTR {
			continue
		}

		if err != nil {
			metric.Count("p2p.cache.store.values.error", 1, []string{"reason", "epoll_wait_error"})
			log.Error().Msgf("epoll wait error: %v", err)
			continue
		}

		// Drain all the pending messages in the queue
		for {
			// Read the next packet from the socket
			numBytesInPacket, _, err := unix.Recvfrom(c.fd, buf, 0)

			// EAGAIN/EWOULDBLOCK: No more messages in the queue, break and wait for next epoll event
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}

			if err != nil {
				metric.Count("p2p.cache.store.values.error", 1, []string{"reason", "recvfrom_error"})
				log.Error().Msgf("Error in recvfrom: %v", err)
				break
			}

			// Make a copy before sending since buf is reused
			responsePacket := make([]byte, numBytesInPacket)
			copy(responsePacket, buf[:numBytesInPacket])
			select {
			case c.outputChannel <- responsePacket:
			default:
				metric.Count("p2p.cache.store.values.error", 1, []string{"reason", "output_channel_full"})
			}
		}
	}
}

func (c *LinuxUDPClient) SendMessage(message []byte, ip string) error {
	if len(message) > c.maxPacketSizeInBytes {
		metric.Count("p2p.cache.store.values.error", 1, []string{"reason", "send_message_greater_than_max_packet_size"})
		return fmt.Errorf("message size is greater than max packet size: %d > %d", len(message), c.maxPacketSizeInBytes)
	}

	serverAddress, err := c.getAddr(ip)
	if err != nil {
		log.Error().Msgf("Error getting destination addr: %v", err)
		return err
	}
	return unix.Sendto(c.fd, message, 0, serverAddress)
}

func (c *LinuxUDPClient) resolveIp(ip string) ([]byte, error) {
	if parsed := net.ParseIP(ip); parsed != nil {
		if v4 := parsed.To4(); v4 != nil {
			return v4, nil
		}
	}

	// resolve dns if ip is not an ipv4 address, useful for resolving ip for local testing using docker dns
	addrs, err := net.LookupIP(ip)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host %s: %w", ip, err)
	}
	for _, a := range addrs {
		if v4 := a.To4(); v4 != nil {
			return v4, nil
		}
	}
	return nil, fmt.Errorf("no IPv4 address found for host %s", ip)
}

func (c *LinuxUDPClient) Close() error {
	if c.fd > 0 {
		return unix.Close(c.fd)
	}
	return nil
}

func (c *LinuxUDPClient) getAddr(host string) (*unix.SockaddrInet4, error) {
	key := host + ":" + strconv.Itoa(c.serverPort)

	addr, ok := c.addrCache.Get(key)
	if ok {
		return addr, nil
	}

	ipv4, err := c.resolveIp(host)
	if err != nil {
		return nil, err
	}
	sa := &unix.SockaddrInet4{Port: c.serverPort}
	copy(sa.Addr[:], ipv4)

	c.addrCache.Set(key, sa, addrTTL+time.Duration(rand.Intn(1000))*time.Second)
	return sa, nil
}
