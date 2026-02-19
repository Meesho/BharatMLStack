package etcd

import (
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type ClientConfig struct {
	Endpoints []string
	Username  string
	Password  string
	Timeout   time.Duration
}

type Client struct {
	cfg ClientConfig
	raw *clientv3.Client
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one ETCD_ENDPOINTS value is required")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	raw, err := clientv3.New(clientv3.Config{
		Endpoints:           cfg.Endpoints,
		Username:            cfg.Username,
		Password:            cfg.Password,
		DialTimeout:         timeout,
		DialKeepAliveTime:   timeout,
		PermitWithoutStream: true,
	})
	if err != nil {
		return nil, err
	}

	return &Client{cfg: cfg, raw: raw}, nil
}

func (c *Client) Endpoints() []string {
	return append([]string(nil), c.cfg.Endpoints...)
}

func (c *Client) Raw() *clientv3.Client {
	return c.raw
}

func (c *Client) Close() error {
	if c.raw == nil {
		return nil
	}
	return c.raw.Close()
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
