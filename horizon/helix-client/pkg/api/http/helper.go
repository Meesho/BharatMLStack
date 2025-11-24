package http

import (
	"fmt"
	"net/http"
)

const (
	Timeout                   = "_TIMEOUT_IN_MS"
	Host                      = "_HOST"
	Port                      = "_PORT"
	DialTimeout               = "_DIAL_TIMEOUT_IN_MS"
	KeepAliveTimeout          = "_KEEP_ALIVE_TIMEOUT_IN_MS"
	MaxIdleConnections        = "_MAX_IDLE_CONNS"
	MaxIdleConnectionsPerHost = "_MAX_IDLE_CONNS_PER_HOST"
	IdleConnectionTimeout     = "_IDLE_CONN_TIMEOUT_IN_MS"
)

// BuildHttpUrl builds a http url from the given host, port and path
func BuildHttpUrl(host string, port int, path string) string {
	return fmt.Sprintf("http://%s:%d:%s", host, port, path)
}

func IsStandard2xx(code int) bool {
	return code >= 200 && code < 300 && http.StatusText(code) != ""
}

func IsStandard3xx(code int) bool {
	return code >= 300 && code < 400 && http.StatusText(code) != ""
}

func IsStandard4xx(code int) bool {
	return code >= 400 && code < 500 && http.StatusText(code) != ""
}

func IsStandard5xx(code int) bool {
	return code >= 500 && code < 600 && http.StatusText(code) != ""
}
