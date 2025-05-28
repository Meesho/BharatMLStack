package config

import (
	"errors"
	"github.com/spf13/viper"
)

const (
	topic                      = "_TOPIC"
	bootstrapURLs              = "_BOOTSTRAP_SERVERS"
	basicAuthCredentialsSource = "_BASIC_AUTH_CREDENTIAL_SOURCE"
	saslUsername               = "_SASL_USERNAME"
	saslPassword               = "_SASL_PASSWORD"
	saslMechanism              = "_SASL_MECHANISM"
	securityProtocol           = "_SECURITY_PROTOCOL"
	groupID                    = "_GROUP_ID"
	autoOffsetReset            = "_AUTO_OFFSET_RESET"
	autoCommitEnable           = "_ENABLE_AUTO_COMMIT"
	reBalanceEnable            = "_RE_BALANCE_ENABLE"
	autoCommitIntervalMs       = "_AUTO_COMMIT_INTERVAL_MS"
	concurrency                = "_LISTENER_CONCURRENCY"
	clientId                   = "_CLIENT_ID"
	batchSize                  = "_BATCH_SIZE"
	pollTimeout                = "_POLL_TIMEOUT"
)

type KafkaConfig struct {
	BootstrapURLs              string
	BasicAuthCredentialsSource string
	SaslUsername               string
	SaslPassword               string
	SaslMechanism              string
	SecurityProtocol           string
	GroupID                    string
	ClientID                   string
	Topic                      string
	AutoOffsetReset            string
	AutoCommitIntervalInMs     int
	AutoCommitEnable           bool
	ReBalanceEnable            bool
	Concurrency                int
	BatchSize                  int
	PollTimeout                int
}

type KafkaConfigGeneratorV1 struct{}

type KafkaConfigGenerator interface {
	BuildConfigFromEnv(envPrefix string) (*KafkaConfig, error)
}

func NewKafkaConfig() KafkaConfigGenerator {
	return &KafkaConfigGeneratorV1{}
}

func (k *KafkaConfigGeneratorV1) BuildConfigFromEnv(envPrefix string) (*KafkaConfig, error) {

	if !viper.IsSet(envPrefix + topic) {
		return nil, errors.New(envPrefix + topic + " not set")
	}
	if !viper.IsSet(envPrefix + bootstrapURLs) {
		return nil, errors.New(envPrefix + bootstrapURLs + " not set")
	}
	if !viper.IsSet(envPrefix + basicAuthCredentialsSource) {
		return nil, errors.New(envPrefix + basicAuthCredentialsSource + " not set")
	}
	if !viper.IsSet(envPrefix + groupID) {
		return nil, errors.New(envPrefix + groupID + " not set")
	}
	if !viper.IsSet(envPrefix + autoOffsetReset) {
		return nil, errors.New(envPrefix + autoOffsetReset + " not set")
	}
	if !viper.IsSet(envPrefix + autoCommitIntervalMs) {
		return nil, errors.New(envPrefix + autoCommitIntervalMs + " not set")
	}
	if !viper.IsSet(envPrefix + concurrency) {
		return nil, errors.New(envPrefix + concurrency + " not set")
	}
	if !viper.IsSet(envPrefix + clientId) {
		return nil, errors.New(envPrefix + clientId + " not set")
	}
	if !viper.IsSet(envPrefix + batchSize) {
		return nil, errors.New(envPrefix + batchSize + " not set")
	}
	if !viper.IsSet(envPrefix + pollTimeout) {
		return nil, errors.New(envPrefix + pollTimeout + " not set")
	}

	return &KafkaConfig{
		Topic:                      viper.GetString(envPrefix + topic),
		BootstrapURLs:              viper.GetString(envPrefix + bootstrapURLs),
		BasicAuthCredentialsSource: viper.GetString(envPrefix + basicAuthCredentialsSource),
		SaslUsername:               viper.GetString(envPrefix + saslUsername),
		SaslPassword:               viper.GetString(envPrefix + saslPassword),
		SaslMechanism:              viper.GetString(envPrefix + saslMechanism),
		SecurityProtocol:           viper.GetString(envPrefix + securityProtocol),
		GroupID:                    viper.GetString(envPrefix + groupID),
		AutoOffsetReset:            viper.GetString(envPrefix + autoOffsetReset),
		AutoCommitIntervalInMs:     viper.GetInt(envPrefix + autoCommitIntervalMs),
		AutoCommitEnable:           viper.GetBool(envPrefix + autoCommitEnable),
		ReBalanceEnable:            viper.GetBool(envPrefix + reBalanceEnable),
		Concurrency:                viper.GetInt(envPrefix + concurrency),
		ClientID:                   viper.GetString(envPrefix + clientId),
		BatchSize:                  viper.GetInt(envPrefix + batchSize),
		PollTimeout:                viper.GetInt(envPrefix + pollTimeout),
	}, nil
}