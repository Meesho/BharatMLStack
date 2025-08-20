package client

// Client defines the interface for UDP communication
type Client interface {
	SendMessage(message []byte, ip string) error
}
