package httpclient

import (
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	httpHelper "github.com/Meesho/BharatMLStack/skye/pkg/api/http"
	"github.com/Meesho/BharatMLStack/skye/pkg/circuitbreaker"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	defaultDialTimeout      = 30000 // in milliseconds
	defaultKeepAliveTimeout = 30000 // in milliseconds
	defaultScheme           = "http"
	defaultPort             = "80"
)

type Config struct {
	Scheme      string
	Host        string
	Port        string
	TimeoutInMs int
	CBConfig    *circuitbreaker.Config
	Transport   *TransportConfig
}

type TransportConfig struct {
	DialTimeoutInMs      int
	MaxIdleConns         int
	MaxIdleConnsPerHost  int
	IdleConnTimeoutInMs  int
	KeepAliveTimeoutInMs int
}

type HTTPClient struct {
	CoreClient     *http.Client
	Endpoint       string
	envPrefix      string
	circuitBreaker circuitbreaker.CircuitBreaker[*http.Request, *http.Response]
}

type pathPattern struct {
	regex       *regexp.Regexp
	replacement string
}

var patterns = []pathPattern{
	{
		regex:       regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`),
		replacement: "/{uuid}",
	},
	{
		regex:       regexp.MustCompile(`/[0-9a-fA-F]{24}`),
		replacement: "/{objectId}",
	},
	{
		regex:       regexp.MustCompile(`/\d+`),
		replacement: "/{id}",
	},
}

func NewConn(envPrefix string) *HTTPClient {
	config := getConfig(envPrefix)
	coreClient := getHTTPClient(config)
	cbConfig := circuitbreaker.BuildConfig(envPrefix)
	var cb circuitbreaker.CircuitBreaker[*http.Request, *http.Response] = nil
	if cbConfig != nil && cbConfig.Enabled {
		cb = circuitbreaker.GetCircuitBreaker[*http.Request, *http.Response](cbConfig)
	}
	return &HTTPClient{
		CoreClient:     coreClient,
		Endpoint:       getEndPoint(config),
		envPrefix:      envPrefix,
		circuitBreaker: cb,
	}
}

func NewConnFromConfig(config *Config, envPrefix string) *HTTPClient {
	coreClient := getHTTPClient(config)
	var cb circuitbreaker.CircuitBreaker[*http.Request, *http.Response] = nil
	if config.CBConfig != nil && config.CBConfig.Enabled {
		cb = circuitbreaker.GetCircuitBreaker[*http.Request, *http.Response](config.CBConfig)
	}
	return &HTTPClient{
		CoreClient:     coreClient,
		Endpoint:       getEndPoint(config),
		envPrefix:      envPrefix,
		circuitBreaker: cb,
	}
}

func getConfig(envPrefix string) *Config {
	config := &Config{
		Scheme: defaultScheme,
		Port:   defaultPort,
	}
	if !viper.IsSet(envPrefix + httpHelper.Host) {
		log.Panic().Msg(envPrefix + httpHelper.Host + " not set")
	}
	if !viper.IsSet(envPrefix + httpHelper.Timeout) {
		log.Panic().Msg(envPrefix + httpHelper.Timeout + " not set")
	}
	config.Host = viper.GetString(envPrefix + httpHelper.Host)
	config.TimeoutInMs = viper.GetInt(envPrefix + httpHelper.Timeout)

	if viper.IsSet(envPrefix + httpHelper.Port) {
		config.Port = viper.GetString(envPrefix + httpHelper.Port)
	}
	config.Transport = getHttpTransportConfig(envPrefix)
	return config
}

func getHttpTransportConfig(envPrefix string) *TransportConfig {
	dialTimeout := defaultDialTimeout
	if viper.IsSet(envPrefix + httpHelper.DialTimeout) {
		dialTimeout = viper.GetInt(envPrefix + httpHelper.DialTimeout)
	}
	keepAliveTimeout := defaultKeepAliveTimeout
	if viper.IsSet(envPrefix + httpHelper.KeepAliveTimeout) {
		keepAliveTimeout = viper.GetInt(envPrefix + httpHelper.KeepAliveTimeout)
	}
	if !viper.IsSet(envPrefix + httpHelper.MaxIdleConnections) {
		log.Panic().Msg(envPrefix + httpHelper.MaxIdleConnections + " not set")
	}
	if !viper.IsSet(envPrefix + httpHelper.MaxIdleConnectionsPerHost) {
		log.Panic().Msg(envPrefix + httpHelper.MaxIdleConnectionsPerHost + " not set")
	}
	if !viper.IsSet(envPrefix + httpHelper.IdleConnectionTimeout) {
		log.Panic().Msg(envPrefix + httpHelper.IdleConnectionTimeout + " not set")
	}

	return &TransportConfig{
		DialTimeoutInMs:      dialTimeout,
		KeepAliveTimeoutInMs: keepAliveTimeout,
		MaxIdleConns:         viper.GetInt(envPrefix + httpHelper.MaxIdleConnections),
		MaxIdleConnsPerHost:  viper.GetInt(envPrefix + httpHelper.MaxIdleConnectionsPerHost),
		IdleConnTimeoutInMs:  viper.GetInt(envPrefix + httpHelper.IdleConnectionTimeout),
	}
}

func getEndPoint(config *Config) string {
	return config.Scheme + "://" + config.Host + ":" + config.Port
}

func getHTTPClient(config *Config) *http.Client {
	log.Debug().Msgf("Creating http client with config: %+v", config)
	return &http.Client{
		Transport: otelhttp.NewTransport(getHttpTransportFromConfig(config.Transport)),
		Timeout:   time.Duration(config.TimeoutInMs) * time.Millisecond,
	}
}

func getHttpTransportFromConfig(transport *TransportConfig) *http.Transport {
	transporter := http.DefaultTransport.(*http.Transport).Clone()
	transporter.DialContext = (&net.Dialer{
		Timeout:   time.Duration(transport.DialTimeoutInMs) * time.Millisecond,
		KeepAlive: time.Duration(transport.KeepAliveTimeoutInMs) * time.Millisecond,
	}).DialContext
	transporter.MaxIdleConns = transport.MaxIdleConns
	transporter.MaxIdleConnsPerHost = transport.MaxIdleConnsPerHost
	transporter.IdleConnTimeout = time.Duration(transport.IdleConnTimeoutInMs) * time.Millisecond
	log.Debug().Msgf("Creating http transporter with config: %+v", transporter)
	return transporter
}

// Do is a wrapper around http.Client.Do and capable to generate metric for external http service
func (h *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	if h.circuitBreaker == nil {
		return h.do(req)
	}
	return h.doWithCB(req)
}

// Do is a wrapper around http.Client.Do and capable to generate metric for external http service
func (h *HTTPClient) do(req *http.Request) (*http.Response, error) {
	startTime := time.Now()
	resp, err := h.CoreClient.Do(req)
	if resp == nil {
		if os.IsTimeout(err) {
			log.Error().Err(err).Msg("Request timed out")
			h.emitMetrics(req, startTime, http.StatusGatewayTimeout)
			return nil, err
		}
		//keeping this 0 as status code as we are not able to get the status code from error
		h.emitMetrics(req, startTime, 0)
		return nil, err
	}
	h.emitMetrics(req, startTime, resp.StatusCode)
	return resp, err
}

// DoWithCB is a wrapper around http.Client.Do and capable to generate metric for external http service
func (h *HTTPClient) doWithCB(req *http.Request) (*http.Response, error) {
	startTime := time.Now()
	resp, err := h.circuitBreaker.Execute(req, h.CoreClient.Do)
	if resp == nil {
		if os.IsTimeout(err) {
			log.Error().Err(err).Msg("Request timed out")
			h.emitMetrics(req, startTime, http.StatusGatewayTimeout)
			return nil, err
		}
		//keeping this 0 as status code as we are not able to get the status code from error
		h.emitMetrics(req, startTime, 0)
		return nil, err
	}
	h.emitMetrics(req, startTime, resp.StatusCode)
	return resp, err
}

func (h *HTTPClient) emitMetrics(req *http.Request, startTime time.Time, statusCode int) {
	latency := time.Since(startTime)
	genericPath := getNormalizedPath(req.URL.Path)
	latencyTags := metric.BuildExternalHTTPServiceLatencyTags(h.envPrefix, genericPath, req.Method, statusCode)
	countTags := metric.BuildExternalHTTPServiceCountTags(h.envPrefix, genericPath, req.Method, statusCode)
	metric.Timing(metric.ExternalApiRequestLatency, latency, latencyTags)
	metric.Incr(metric.ExternalApiRequestCount, countTags)
}

func getNormalizedPath(path string) string {
	normalizedPath := path
	for _, pattern := range patterns {
		normalizedPath = pattern.regex.ReplaceAllString(normalizedPath, pattern.replacement)
	}
	return normalizedPath
}
