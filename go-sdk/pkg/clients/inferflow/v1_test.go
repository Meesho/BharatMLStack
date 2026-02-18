package inferflow

import (
	"testing"

	"github.com/spf13/viper"
)

func TestGetClientConfigs_Valid(t *testing.T) {
	viper.Set("INFERFLOW_CLIENT_V1_HOST", "localhost")
	viper.Set("INFERFLOW_CLIENT_V1_PORT", "8080")
	viper.Set("INFERFLOW_CLIENT_V1_DEADLINE_MS", 500)
	viper.Set("INFERFLOW_CLIENT_V1_PLAINTEXT", true)
	defer viper.Reset()

	conf, err := getClientConfigs(V1Prefix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", conf.Host)
	}
	if conf.Port != "8080" {
		t.Errorf("expected port 8080, got %s", conf.Port)
	}
	if conf.DeadlineExceedMS != 500 {
		t.Errorf("expected deadline 500, got %d", conf.DeadlineExceedMS)
	}
	if !conf.PlainText {
		t.Error("expected plaintext true")
	}
}

func TestGetClientConfigs_MissingHost(t *testing.T) {
	viper.Reset()
	viper.Set("INFERFLOW_CLIENT_V1_PORT", "8080")
	viper.Set("INFERFLOW_CLIENT_V1_DEADLINE_MS", 500)
	defer viper.Reset()

	_, err := getClientConfigs(V1Prefix)
	if err == nil {
		t.Fatal("expected error for missing host, got nil")
	}
}

func TestGetClientConfigs_EmptyPort(t *testing.T) {
	viper.Reset()
	viper.Set("INFERFLOW_CLIENT_V1_HOST", "localhost")
	viper.Set("INFERFLOW_CLIENT_V1_PORT", "")
	viper.Set("INFERFLOW_CLIENT_V1_DEADLINE_MS", 500)
	defer viper.Reset()

	_, err := getClientConfigs(V1Prefix)
	if err == nil {
		t.Fatal("expected error for empty port, got nil")
	}
}

func TestGetClientConfigs_InvalidDeadline(t *testing.T) {
	viper.Set("INFERFLOW_CLIENT_V1_HOST", "localhost")
	viper.Set("INFERFLOW_CLIENT_V1_PORT", "8080")
	viper.Set("INFERFLOW_CLIENT_V1_DEADLINE_MS", 0)
	defer viper.Reset()

	_, err := getClientConfigs(V1Prefix)
	if err == nil {
		t.Fatal("expected error for zero deadline, got nil")
	}
}

func TestGetClientConfigs_Defaults(t *testing.T) {
	viper.Set("INFERFLOW_CLIENT_V1_HOST", "inferflow.svc")
	defer viper.Reset()

	conf, err := getClientConfigs(V1Prefix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conf.Port != DefaultPort {
		t.Errorf("expected default port %s, got %s", DefaultPort, conf.Port)
	}
	if conf.DeadlineExceedMS != DefaultDeadlineMS {
		t.Errorf("expected default deadline %d, got %d", DefaultDeadlineMS, conf.DeadlineExceedMS)
	}
	if conf.PlainText != DefaultPlainText {
		t.Errorf("expected default plaintext %v, got %v", DefaultPlainText, conf.PlainText)
	}
}

func TestValidConfigs(t *testing.T) {
	tests := []struct {
		name    string
		config  *ClientConfig
		wantOK  bool
	}{
		{
			name:   "valid config",
			config: &ClientConfig{Host: "localhost", Port: "8080", DeadlineExceedMS: 200},
			wantOK: true,
		},
		{
			name:   "empty host",
			config: &ClientConfig{Host: "", Port: "8080", DeadlineExceedMS: 200},
			wantOK: false,
		},
		{
			name:   "empty port",
			config: &ClientConfig{Host: "localhost", Port: "", DeadlineExceedMS: 200},
			wantOK: false,
		},
		{
			name:   "zero deadline",
			config: &ClientConfig{Host: "localhost", Port: "8080", DeadlineExceedMS: 0},
			wantOK: false,
		},
		{
			name:   "negative deadline",
			config: &ClientConfig{Host: "localhost", Port: "8080", DeadlineExceedMS: -1},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := validConfigs(tt.config)
			if ok != tt.wantOK {
				t.Errorf("validConfigs() = %v, want %v, err: %v", ok, tt.wantOK, err)
			}
		})
	}
}

func TestGetInferflowClient_InvalidVersion(t *testing.T) {
	c := GetInferflowClient(99)
	if c != nil {
		t.Error("expected nil for unsupported version")
	}
}

func TestGetInferflowClientFromConfig_InvalidVersion(t *testing.T) {
	c := GetInferflowClientFromConfig(99, ClientConfig{}, "test")
	if c != nil {
		t.Error("expected nil for unsupported version")
	}
}
