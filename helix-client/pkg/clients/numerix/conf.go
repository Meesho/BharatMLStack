package numerix

import (
	"encoding/json"
	"fmt"
)

const (
	Host       = "HOST"
	Port       = "PORT"
	DeadlineMS = "DEADLINE_MS"
	PlainText  = "PLAINTEXT"
	AuthToken  = "AUTH_TOKEN"
	BatchSize  = "BATCH_SIZE"
)

type ClientConfig struct {
	Host             string `json:"Host"`
	Port             string `json:"Port"`
	DeadlineExceedMS int    `json:"DeadlineExceedMS"`
	PlainText        bool   `json:"PlainText"`
	AuthToken        string `json:"AuthToken"`
	BatchSize        int    `json:"BatchSize"`
	CallerId         string `json:"CallerId"`
}

func getClientConfigs(configBytes []byte) (*ClientConfig, error) {
	conf := &ClientConfig{}

	err := json.Unmarshal(configBytes, &conf)
	if err != nil {
		return nil, err
	}

	if valid, err := validConfigs(conf); !valid {
		return nil, err
	}

	return conf, nil
}

func validConfigs(configs *ClientConfig) (bool, error) {
	if configs.Host == "" {
		return false, fmt.Errorf("numerix service host is invalid, configured value: %v", configs.Host)
	}
	if configs.Port == "" {
		return false, fmt.Errorf("numerix service port is invalid, configured value: %v", configs.Port)
	}
	if configs.DeadlineExceedMS <= 0 {
		return false, fmt.Errorf("numerix service deadline exceed timeout is invalid, configured value: %v",
			configs.DeadlineExceedMS)
	}
	if configs.AuthToken == "" {
		return false, fmt.Errorf("numerix service auth token not configured")
	}
	if configs.BatchSize <= 0 {
		return false, fmt.Errorf("numerix service batch size is invalid, configured value: %v", configs.BatchSize)
	}
	if configs.CallerId == "" {
		return false, fmt.Errorf("numerix service caller id not configured")
	}
	return true, nil
}
