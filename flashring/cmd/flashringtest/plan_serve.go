package main

// plan_serve runs flashring as a long-lived process so the standalone
// e2e-test-kit (`stk run test/service-tests/flashringtest/`) can drive it
// over the network. flashring ships as a Go library and has no production
// server; this plan exists *only* so kit-driven smokes have a SUT to talk
// to. The plan exposes two sibling listeners over a single WrapCache:
//
//   - RESP server on -resp-port (default 6379): the production-relevant
//     public API documented in CLAUDE.md ("Never introduce backward-
//     incompatible changes to ... the RESP wire surface"). Today no `stk`
//     action handler speaks RESP; it stays running so future kit upgrades
//     or `exec`-based redis-cli drives can target it without a code
//     change.
//
//   - HTTP wrapper on -http-port (default 8081): minimal Put/Get/Healthz
//     endpoints used by the kit's `http` action. Test-only.
//
// stdlib net/http is used (not Gin) because the binary already imports
// net/http for pprof and this surface is testing-only — CLAUDE.md's Gin
// mandate is for production server components.

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/Meesho/BharatMLStack/flashring/internal/server"
	"github.com/rs/zerolog/log"
)

const errKeyRequired = "key is required"

// assertMountUnderTempDir refuses to proceed if mountPoint is not under
// os.TempDir. NewWrapCache wipes every file at mountPoint on init and the
// serve plan also clears it on shutdown; restricting to TempDir guards
// against accidentally pointing the serve plan at /home, /etc, or a real
// NVMe mount and losing data. Override via TMPDIR if a different scratch
// root is needed.
func assertMountUnderTempDir(mountPoint string) {
	cleanMount, err := filepath.Abs(filepath.Clean(mountPoint))
	if err != nil {
		log.Fatal().Err(err).Str("mount", mountPoint).Msg("resolve mount path")
	}
	tempRoot := filepath.Clean(os.TempDir()) + string(os.PathSeparator)
	if !strings.HasPrefix(cleanMount+string(os.PathSeparator), tempRoot) {
		log.Fatal().Str("mount", cleanMount).Str("temp-root", tempRoot).
			Msg("unsafe mount: serve plan only accepts paths under os.TempDir (override via TMPDIR)")
	}
}

func planServe() {
	mountDefault := os.Getenv("FLASHRING_TEST_MOUNT")
	if mountDefault == "" {
		mountDefault = os.TempDir() + "/flashring-serve"
	}
	httpPortDefault := envInt("FLASHRING_HTTP_PORT", 8081)
	respPortDefault := envInt("FLASHRING_RESP_PORT", 6379)

	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	mountPoint := fs.String("mount", mountDefault, "data directory for shard files")
	httpPort := fs.Int("http-port", httpPortDefault, "port for the test-only HTTP wrapper")
	respPort := fs.Int("resp-port", respPortDefault, "port for the RESP server")
	_ = fs.Parse(os.Args[1:])

	assertMountUnderTempDir(*mountPoint)

	if err := os.MkdirAll(*mountPoint, 0o755); err != nil {
		log.Fatal().Err(err).Str("mount", *mountPoint).Msg("create mount")
	}

	// Conservative single-shard config — matches plan_smoke's shape. The kit
	// smoke exercises one Put + one Get, so anything bigger is waste.
	const (
		memtableSize = 1 << 20 // 1 MiB
		fileSize     = 4 << 20 // 4 MiB
	)
	cfg := cachepkg.Config{
		NumShards:             1,
		KeysPerShard:          1024,
		FileSize:              fileSize,
		MemtableSize:          memtableSize,
		ReWriteScoreThreshold: 0.5,
		GridSearchEpsilon:     0.01,
		SampleDuration:        5 * time.Second,
		FreqBands:             []int{10, 100, 1000},
		RecencyBands:          []int{1000, 100, 10},
	}

	wc, err := cachepkg.NewWrapCache(cfg, *mountPoint)
	if err != nil {
		log.Fatal().Err(err).Str("mount", *mountPoint).Msg("NewWrapCache")
	}

	respLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *respPort))
	if err != nil {
		log.Fatal().Err(err).Int("port", *respPort).Msg("listen RESP")
	}
	go func() {
		log.Info().Int("port", *respPort).Msg("RESP server listening")
		if err := server.Serve(respLn, wc); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Error().Err(err).Msg("RESP server exited")
		}
	}()

	httpSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *httpPort),
		Handler:           newHTTPHandler(wc),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Info().Int("port", *httpPort).Msg("HTTP wrapper listening")
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP wrapper exited")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	sig := <-stop
	log.Info().Str("signal", sig.String()).Msg("shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Warn().Err(err).Msg("HTTP wrapper shutdown")
	}
	_ = respLn.Close()
	if err := wc.Close(); err != nil {
		log.Warn().Err(err).Msg("cache close")
	}
	if files, err := os.ReadDir(*mountPoint); err == nil {
		for _, f := range files {
			_ = os.Remove(filepath.Join(*mountPoint, f.Name()))
		}
	}
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func newHTTPHandler(wc *cachepkg.WrapCache) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("POST /put", handlePut(wc))

	mux.HandleFunc("GET /get", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			writeJSONError(w, http.StatusBadRequest, errKeyRequired)
			return
		}
		val, found, expired := wc.Get(key)
		writeJSON(w, http.StatusOK, map[string]any{
			"found":   found,
			"expired": expired,
			"value":   string(val),
		})
	})

	mux.HandleFunc("DELETE /delete", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			writeJSONError(w, http.StatusBadRequest, errKeyRequired)
			return
		}
		existed := wc.Delete(key)
		writeJSON(w, http.StatusOK, map[string]any{"existed": existed})
	})

	mux.HandleFunc("POST /mget", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Keys []string `json:"keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "decode body: "+err.Error())
			return
		}
		results := wc.MGet(req.Keys)
		out := make([]map[string]any, len(results))
		for i, r := range results {
			out[i] = map[string]any{
				"found":   r.Found,
				"expired": r.Expired,
				"value":   string(r.Value),
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": out})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func handlePut(wc *cachepkg.WrapCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
			TTLMS int64  `json:"ttl_ms"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "decode body: "+err.Error())
			return
		}
		if req.Key == "" {
			writeJSONError(w, http.StatusBadRequest, errKeyRequired)
			return
		}
		if req.TTLMS < 0 {
			writeJSONError(w, http.StatusBadRequest, "ttl_ms must be >= 0")
			return
		}
		ttl := time.Duration(req.TTLMS) * time.Millisecond
		if err := wc.Put(req.Key, []byte(req.Value), ttl); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
