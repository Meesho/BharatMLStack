package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Env struct {
	AppPort         int
	AppName         string
	AppLogLevel     string
	AppEnv          string
	EtcdEndpoints   []string
	EtcdUsername    string
	EtcdPassword    string
	EtcdTimeout     time.Duration
	UseMockAdapters bool
}

var (
	once      sync.Once
	instance  Env
	initError error
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

	endpoints := []string{"127.0.0.1:2379"}
	if raw := strings.TrimSpace(os.Getenv("ETCD_ENDPOINTS")); raw != "" {
		parts := strings.Split(raw, ",")
		endpoints = make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				endpoints = append(endpoints, p)
			}
		}
	}

	return Env{
		AppPort:         port,
		AppName:         strings.TrimSpace(os.Getenv("APP_NAME")),
		AppLogLevel:     strings.TrimSpace(os.Getenv("APP_LOG_LEVEL")),
		AppEnv:          strings.TrimSpace(os.Getenv("APP_ENV")),
		EtcdEndpoints:   endpoints,
		EtcdUsername:    strings.TrimSpace(os.Getenv("ETCD_USERNAME")),
		EtcdPassword:    os.Getenv("ETCD_PASSWORD"),
		EtcdTimeout:     timeout,
		UseMockAdapters: useMock,
	}, nil
}

func InitEnv() error {
	once.Do(func() {
		viper.AutomaticEnv()
		instance, initError = Load()
	})
	return initError
}

func Instance() Env {
	if err := InitEnv(); err != nil {
		panic(err)
	}
	return instance
}
