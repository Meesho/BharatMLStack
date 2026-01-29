package skye

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	Host              = "HOST"
	Port              = "PORT"
	DeadlineMS        = "DEADLINE_MS"
	PlainText         = "PLAINTEXT"
	AuthToken         = "AUTH_TOKEN"
	DefaultHost       = ""
	DefaultPort       = "8080"
	DefaultDeadlineMS = 200
	DefaultPlainText  = true
	DefaultAuthToken  = ""
)

type ClientConfig struct {
	Host             string
	Port             string
	DeadlineExceedMS int
	PlainText        bool
	AuthToken        string
}

func getClientConfigs(prefix string) (*ClientConfig, error) {
	host := DefaultHost
	port := DefaultPort
	deadline := DefaultDeadlineMS
	plaintext := DefaultPlainText
	authToken := DefaultAuthToken

	if viper.IsSet(prefix + Host) {
		host = viper.GetString(prefix + Host)
	}
	if viper.IsSet(prefix + Port) {
		port = viper.GetString(prefix + Port)
	}
	if viper.IsSet(prefix + DeadlineMS) {
		deadline = viper.GetInt(prefix + DeadlineMS)
	}
	if viper.IsSet(prefix + PlainText) {
		plaintext = viper.GetBool(prefix + PlainText)
	}
	if viper.IsSet(prefix + AuthToken) {
		authToken = viper.GetString(prefix + AuthToken)
	}
	conf := &ClientConfig{
		Host:             host,
		Port:             port,
		DeadlineExceedMS: deadline,
		PlainText:        plaintext,
		AuthToken:        authToken,
	}
	if valid, err := validConfigs(conf); !valid {
		return nil, err
	}
	return conf, nil
}

func validConfigs(configs *ClientConfig) (bool, error) {
	if configs.Host == "" {
		return false, fmt.Errorf("skye service host is invalid, configured value: %v", configs.Host)
	}
	if configs.Port == "" {
		return false, fmt.Errorf("skye service port is invalid, configured value: %v", configs.Port)
	}
	if configs.DeadlineExceedMS <= 0 {
		return false, fmt.Errorf("skye service deadline exceed timeout is invalid, configured value: %v",
			configs.DeadlineExceedMS)
	}
	if configs.AuthToken == "" {
		return false, fmt.Errorf("skye service auth token not configured")
	}
	return true, nil
}
