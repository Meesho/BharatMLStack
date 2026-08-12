package profiling

import (
	"net/http/pprof"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/rs/zerolog/log"
)

// mountPprof attaches the selected pprof endpoints to the metrics server's mux,
// so they are served on the same port as /metrics rather than a second port.
func mountPprof(handlers []PprofHandler) {
	mux := metric.Mux()
	for _, h := range handlers {
		switch h {
		case PprofHeap:
			mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		case PprofAllocs:
			mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		case PprofGoroutine:
			mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		case PprofThreadcreate:
			mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		case PprofBlock:
			mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		case PprofMutex:
			mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		case PprofProfile:
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		case PprofTrace:
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		case PprofCmdline:
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		case PprofSymbol:
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		default:
			log.Warn().Int("handler", int(h)).Msg("profiling: unknown PprofHandler, skipping")
		}
	}
}
