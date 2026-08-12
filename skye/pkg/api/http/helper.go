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

// BuildHttpUrl builds a http url from the given host, port and path.
//
// The separator between port and path is deliberately absent: path already
// carries its own leading "/". An earlier ":%s" here produced
// "http://host:8080:/path", which Go's URL parser accepted leniently before
// 1.26 and rejects from 1.26 on — surfacing as a nil *http.Request rather than
// as a parse error, because the caller below did not check err.
func BuildHttpUrl(host string, port int, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, path)
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
