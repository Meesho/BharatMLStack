package etcd

import (
	"fmt"
	"strings"
	"time"
)

type ClientConfig struct {
	Endpoints []string
	Username  string
	Password  string
	Timeout   time.Duration
}

// Client is a placeholder for real etcd client integration.
// Replace this with go.etcd.io/etcd/client/v3 in the next phase.
type Client struct {
	cfg ClientConfig
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one ETCD_ENDPOINTS value is required")
	}
	return &Client{cfg: cfg}, nil
}

func (c *Client) Endpoints() []string {
	return append([]string(nil), c.cfg.Endpoints...)
}

func ParseEndpoints(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
