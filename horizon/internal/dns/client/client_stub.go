//go:build !meesho

package client

import "fmt"

// DNSClient interface for DNS operations
// This interface is defined in the stub to maintain compatibility
// The actual implementation is in the internal configs repo
type DNSClient interface {
	CreateCloudDNSRecord(appName, bu, ingressClass, priorityV2 string, isServiceGrpc bool, workingEnv string) error
	CreateCoreDNSRecord(appName, bu, ingressClass, priorityV2 string, isServiceGrpc, isMultiZone bool, workingEnv string) error
}

// NewDNSClient creates a new DNS client (stub implementation for open-source builds)
// This is a no-op for open-source builds - DNS functionality is not available
func NewDNSClient() DNSClient {
	return &dnsClientStub{}
}

type dnsClientStub struct{}

func (c *dnsClientStub) CreateCloudDNSRecord(appName, bu, ingressClass, priorityV2 string, isServiceGrpc bool, workingEnv string) error {
	return fmt.Errorf("DNS functionality is not available in open-source builds. Provide organization-specific implementations to enable DNS operations")
}

func (c *dnsClientStub) CreateCoreDNSRecord(appName, bu, ingressClass, priorityV2 string, isServiceGrpc, isMultiZone bool, workingEnv string) error {
	return fmt.Errorf("DNS functionality is not available in open-source builds. Provide organization-specific implementations to enable DNS operations")
}
