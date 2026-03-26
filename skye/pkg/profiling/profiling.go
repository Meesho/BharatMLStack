package profiling

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"net/http"
	_ "net/http/pprof"
	"sync"
)

var (
	once        sync.Once
	initialized = false
)

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
