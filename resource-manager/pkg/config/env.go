package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Env struct {
	AppPort                int
	AppName                string
	AppLogLevel            string
	AppEnv                 string
	APIAuthToken           string
	IdempotencyTTLSeconds  int64
	EtcdEndpoints          []string
	EtcdUsername           string
	EtcdPassword           string
	EtcdTimeout            time.Duration
	UseMockAdapters        bool
	BootstrapEtcdLayout    bool
	BootstrapLayoutTimeout time.Duration
}

var (
	initialized bool
	once        sync.Once
	instance    Env
	initError   error
)

func Load() (Env, error) {
	port := 8080
	if raw := strings.TrimSpace(os.Getenv("APP_PORT")); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p <= 0 {
			return Env{}, fmt.Errorf("invalid APP_PORT: %q", raw)
		}
		port = p
	}

	timeout := 5 * time.Second
	if raw := strings.TrimSpace(os.Getenv("ETCD_TIMEOUT_SECONDS")); raw != "" {
		sec, err := strconv.Atoi(raw)
		if err != nil || sec <= 0 {
			return Env{}, fmt.Errorf("invalid ETCD_TIMEOUT_SECONDS: %q", raw)
		}
		timeout = time.Duration(sec) * time.Second
	}

	useMock := true
	if raw := strings.TrimSpace(os.Getenv("USE_MOCK_ADAPTERS")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Env{}, fmt.Errorf("invalid USE_MOCK_ADAPTERS: %q", raw)
		}
		useMock = v
	}

	bootstrapLayout := true
	if raw := strings.TrimSpace(os.Getenv("BOOTSTRAP_ETCD_LAYOUT")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return Env{}, fmt.Errorf("invalid BOOTSTRAP_ETCD_LAYOUT: %q", raw)
		}
		bootstrapLayout = v
	}

	bootstrapTimeout := 10 * time.Second
	if raw := strings.TrimSpace(os.Getenv("BOOTSTRAP_ETCD_LAYOUT_TIMEOUT_SECONDS")); raw != "" {
		sec, err := strconv.Atoi(raw)
		if err != nil || sec <= 0 {
			return Env{}, fmt.Errorf("invalid BOOTSTRAP_ETCD_LAYOUT_TIMEOUT_SECONDS: %q", raw)
		}
		bootstrapTimeout = time.Duration(sec) * time.Second
	}

	endpoints := []string{"127.0.0.1:2379"}
	if raw := strings.TrimSpace(os.Getenv("ETCD_ENDPOINTS")); raw != "" {
		endpoints = parseEtcdEndpoints(raw)
	} else if raw := strings.TrimSpace(os.Getenv("ETCD_SERVER")); raw != "" {
		endpoints = parseEtcdEndpoints(raw)
	}

	apiAuthToken := strings.TrimSpace(os.Getenv("API_AUTH_TOKEN"))
	if apiAuthToken == "" {
		return Env{}, fmt.Errorf("invalid API_AUTH_TOKEN: cannot be empty")
	}
	idempotencyTTLSeconds := int64(24 * 60 * 60)
	if raw := strings.TrimSpace(os.Getenv("IDEMPOTENCY_TTL_SECONDS")); raw != "" {
		ttl, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || ttl <= 0 {
			return Env{}, fmt.Errorf("invalid IDEMPOTENCY_TTL_SECONDS: %q", raw)
		}
		idempotencyTTLSeconds = ttl
	}

	return Env{
		AppPort:                port,
		AppName:                strings.TrimSpace(os.Getenv("APP_NAME")),
		AppLogLevel:            strings.TrimSpace(os.Getenv("APP_LOG_LEVEL")),
		AppEnv:                 strings.TrimSpace(os.Getenv("APP_ENV")),
		APIAuthToken:           apiAuthToken,
		IdempotencyTTLSeconds:  idempotencyTTLSeconds,
		EtcdEndpoints:          endpoints,
		EtcdUsername:           strings.TrimSpace(os.Getenv("ETCD_USERNAME")),
		EtcdPassword:           os.Getenv("ETCD_PASSWORD"),
		EtcdTimeout:            timeout,
		UseMockAdapters:        useMock,
		BootstrapEtcdLayout:    bootstrapLayout,
		BootstrapLayoutTimeout: bootstrapTimeout,
	}, nil
}

func parseEtcdEndpoints(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func InitEnv() {
	if initialized {
		log.Debug().Msg("Env already initialized!")
		return
	}
	once.Do(func() {
		viper.AutomaticEnv()
		instance, initError = Load()
		if initError != nil {
			log.Panic().Err(initError).Msg("failed to load env")
		}
		initialized = true
		log.Info().Msg("Env initialized!")
	})
}

func Instance() Env {
	InitEnv()
	if initError != nil {
		panic(initError)
	}
	return instance
}
