package grpc

import (
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	hostEnvSuffix          = "_HOST"
	portEnvSuffix          = "_PORT"
	timeoutEnvSuffix       = "_TIMEOUT"
	grpcLBPolicyEnvSuffix  = "_GRPC_LB_POLICY"
	grpcPlainTextEnvSuffix = "_GRPC_PLAIN_TEXT"

	defaultGrpcLBPolicy = "round_robin"
)

type ServerOption struct {
	Host                string `json:"host"`
	Port                string `json:"port"`
	Timeout             int    `json:"timeout_in_ms"`
	IsPlainText         bool   `json:"is_plain_text"`
	LoadBalancingPolicy string `json:"load_balancing_policy"`
}

// BuildServerConfigFromEnv - builds the config for grpc server
func BuildServerConfigFromEnv(envPrefix string) (*ServerOption, error) {
	log.Debug().Msgf("building grpc config from env, env prefix - %s", envPrefix)
	if !viper.IsSet(envPrefix + hostEnvSuffix) {
		return nil, errors.New("Host Id is not present for env prefix - " + envPrefix)
	}
	host := viper.GetString(envPrefix + hostEnvSuffix)

	if !viper.IsSet(envPrefix + portEnvSuffix) {
		return nil, errors.New("Port is not present for env prefix - " + envPrefix)
	}
	port := viper.GetString(envPrefix + portEnvSuffix)

	if !viper.IsSet(envPrefix + timeoutEnvSuffix) {
		return nil, errors.New("timeout is not present for env prefix - " + envPrefix)
	}
	timeout := viper.GetInt(envPrefix + timeoutEnvSuffix)

	var grpcLBPolicy string
	if !viper.IsSet(envPrefix + grpcLBPolicyEnvSuffix) {
		log.Warn().Msgf("grpcLbPolicy is not present for env prefix - %s. Setting it to round-robin", envPrefix)
		grpcLBPolicy = defaultGrpcLBPolicy
	} else {
		grpcLBPolicy = viper.GetString(envPrefix + grpcLBPolicyEnvSuffix)
	}

	grpcPlainText := viper.GetBool(envPrefix + grpcPlainTextEnvSuffix)

	return &ServerOption{
		Host:                host,
		Port:                port,
		Timeout:             timeout,
		IsPlainText:         grpcPlainText,
		LoadBalancingPolicy: grpcLBPolicy,
	}, nil
}
