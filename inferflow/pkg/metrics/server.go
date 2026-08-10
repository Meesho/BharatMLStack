package metrics

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// defaultMetricsServerPort is the port the platform's Prometheus scrape config
// targets. It must stay 14271 for the go-runtime-metrics dashboards to see this
// service.
const defaultMetricsServerPort = 14271

var (
	listener      net.Listener
	metricsServer *http.Server
	mux           *http.ServeMux
	registerer    prometheus.Registerer
)

// Registerer returns the Prometheus registerer, pre-labelled with service and
// env. Nil until Init has run; pkg/profiling registers its collectors here.
func Registerer() prometheus.Registerer { return registerer }

// Mux returns the metrics server's mux, so pprof is served on the same port as
// /metrics. Nil until Init has run.
func Mux() *http.ServeMux { return mux }

// Addr returns the bound listener address, or "" before Init.
func Addr() string {
	if listener == nil {
		return ""
	}
	return listener.Addr().String()
}

// initMetricsServer binds the metrics port and serves /metrics.
//
// appName/appEnv are passed in rather than read from viper because the modules
// source them differently. Binds synchronously so a port clash surfaces here
// rather than inside the serve goroutine. Returns an error instead of panicking:
// this is additive to the existing StatsD path, and a service that cannot start
// the Prometheus endpoint should keep running rather than fail to boot.
func initMetricsServer(port int, appName, appEnv string) error {
	if appName == "" {
		return fmt.Errorf("app name is required for the metrics server")
	}
	if appEnv == "" {
		return fmt.Errorf("app env is required for the metrics server")
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("metrics server listen on :%d: %w", port, err)
	}
	listener = ln

	registry := prometheus.NewRegistry()
	registerer = prometheus.WrapRegistererWith(prometheus.Labels{
		"service": appName,
		"env":     appEnv,
	}, registry)

	mux = http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	metricsServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		// pprof CPU profiles run up to 120s; 3m leaves headroom.
		WriteTimeout: 3 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info(fmt.Sprintf("Starting metrics server addr=%s", ln.Addr().String()))
		if err := metricsServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server stopped", err)
		}
	}()
	return nil
}
