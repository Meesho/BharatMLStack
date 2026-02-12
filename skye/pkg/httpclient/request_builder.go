package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	httpHelper "github.com/Meesho/BharatMLStack/skye/pkg/api/http"
)

type RequestBuilder struct {
	host    string
	port    int
	path    string
	method  string
	headers map[string]string
	body    any
	ctx     context.Context
}

func NewHttpRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		headers: make(map[string]string),
	}
}

// WithHost sets the host for the request
func (h *RequestBuilder) WithHost(host string) *RequestBuilder {
	h.host = host
	return h
}

// WithPort sets the port for the request
func (h *RequestBuilder) WithPort(port int) *RequestBuilder {
	h.port = port
	return h
}

// WithPath sets the path for the request
func (h *RequestBuilder) WithPath(path string) *RequestBuilder {
	h.path = path
	return h
}

// WithMethod sets the method for the request
func (h *RequestBuilder) WithMethod(method string) *RequestBuilder {
	h.method = method
	return h
}

// WithHeader adds the header for the request
func (h *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	h.headers[key] = value
	return h
}

// WithHeaders adds the headers for the request
func (h *RequestBuilder) WithHeaders(headers map[string]string) *RequestBuilder {
	for key, value := range headers {
		h.headers[key] = value
	}
	return h
}

// WithBody sets the body for the request
func (h *RequestBuilder) WithBody(body any) *RequestBuilder {
	h.body = body
	return h
}

// WithContext sets the context for the request
func (h *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	h.ctx = ctx
	return h
}

// BuildContentTypeJson validates the builder request and builds the http request
// with content type as application/json
func (h *RequestBuilder) BuildContentTypeJson() (*http.Request, error) {
	requestBody, err := json.Marshal(h.body)
	if err != nil {
		return nil, err
	}
	if len(h.host) == 0 {
		return nil, errors.New("host is required")
	}
	if h.port == 0 {
		return nil, errors.New("invalid port")
	}
	if len(h.path) == 0 {
		return nil, errors.New("path is required")
	}
	if len(h.method) == 0 {
		return nil, errors.New("method is required")
	}
	if h.ctx == nil {
		return nil, errors.New("context is required, pass context.Background() if not required")
	}
	req, err := http.NewRequestWithContext(h.ctx, h.method, httpHelper.BuildHttpUrl(h.host, h.port, h.path), bytes.NewBuffer(requestBody))
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set(httpHelper.HeaderContentType, httpHelper.HeaderValueApplicationJson)
	return req, err
}
