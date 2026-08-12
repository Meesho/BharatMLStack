package profiling

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	once        sync.Once
	initialized = false

	optsOnce sync.Once
	optsErr  error
)

// InitWithOptions mounts pprof and Go runtime metrics on the metrics server
// started by metric.Init, and optionally starts the continuous profiler.
//
// metric.Init MUST have run first: metric.Mux and metric.Registerer are read
// here, and this returns an error without doing anything when they are nil.
//
// Ordering matches go-core deliberately: pprof mounts and the continuous
// profiler start BEFORE runtime-metric registration, so a registration error
// leaves those two already in effect. Guarded by sync.Once, so a failed first
// call latches its error and later calls will not retry.
//
// Prefer this over Init, which serves pprof on a separate PROFILING_PORT and
// exports no runtime metrics.
func InitWithOptions(opts ...Option) error {
	if len(opts) == 0 {
		return fmt.Errorf("profiling: InitWithOptions requires at least one option")
	}
	optsOnce.Do(func() {
		if metric.Registerer() == nil || metric.Mux() == nil {
			optsErr = fmt.Errorf("profiling: metric.Init() must be called first")
			return
		}

		cfg := &config{}
		for _, opt := range opts {
			opt(cfg)
		}

		if len(cfg.pprofHandlers) > 0 {
			mountPprof(cfg.pprofHandlers)
			log.Info().Msg("pprof endpoints mounted")
		}
		if cfg.continuousProfiler {
			if err := startContinuousProfiler(); err != nil {
				log.Warn().Err(err).Msg("continuous profiler failed to start")
			} else {
				log.Info().Msg("Continuous profiler started")
			}
		}
		if len(cfg.runtimeMetrics) > 0 {
			if err := registerRuntimeMetrics(cfg.runtimeMetrics); err != nil {
				optsErr = fmt.Errorf("profiling: register runtime metrics: %w", err)
				return
			}
			log.Info().Msg("Go runtime metrics registered")
		}
	})
	return optsErr
}

// Deprecated: serves pprof on its own PROFILING_PORT and exports no runtime
// metrics. Use InitWithOptions.
func Init() {
	if !checkProfilingEnabled() {
		return
	}
	if initialized {
		log.Debug().Msg("Profiling environment already initialized!")
		return
	}
	once.Do(func() {
		initializeProfiling()
	})
}

func checkProfilingEnabled() bool {
	if !viper.GetBool("PROFILING_ENABLED") {
		log.Info().Msg("Profiling is not enabled!")
		return false
	}
	return true
}

func initializeProfiling() {
	profilingPort := viper.GetInt("PROFILING_PORT")
	if profilingPort == 0 {
		log.Fatal().Msg("PROFILING_PORT is not set!")
	}

	initProfilingTool(profilingPort)
	initialized = true
	log.Info().Msg("Profiling environment initialized!")
}

func initProfilingTool(port int) {
	go func() {
		addr := fmt.Sprintf(":%d", port)
		log.Info().Msgf("Starting profiling server on %v", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal().Msgf("ListenAndServe error: %v", err)
		}
	}()
}
