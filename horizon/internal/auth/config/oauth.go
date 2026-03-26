package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	oauthConfig     *OAuthConfig
	oauthConfigOnce sync.Once
)

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	RedirectURI        string
	SSOEnabled         bool
	SSOProvider        string
	AccessTokenExpiry  int // in hours, default 24
	RefreshTokenExpiry int // in days, default 7
}

// GetOAuthConfig returns the OAuth configuration singleton
func GetOAuthConfig() *OAuthConfig {
	oauthConfigOnce.Do(func() {
		oauthConfig = &OAuthConfig{
			GoogleClientID:     getEnvOrViper("GOOGLE_OAUTH_CLIENT_ID", ""),
			GoogleClientSecret: getEnvOrViper("GOOGLE_OAUTH_CLIENT_SECRET", ""),
			RedirectURI:        getEnvOrViper("GOOGLE_OAUTH_REDIRECT_URI", "http://localhost:3000/login"),
			SSOEnabled:         getEnvOrViperBool("SSO_ENABLED", true),
			SSOProvider:        getEnvOrViper("SSO_PROVIDER", "google"),     // password, google (use constants in code)
			AccessTokenExpiry:  getEnvOrViperInt("ACCESS_TOKEN_EXPIRY", 24), // hours
			RefreshTokenExpiry: getEnvOrViperInt("REFRESH_TOKEN_EXPIRY", 7), // days
		}

		if oauthConfig.SSOEnabled && oauthConfig.GoogleClientID == "" {
			log.Warn().Msg("SSO is enabled but GOOGLE_OAUTH_CLIENT_ID is not set")
		}
		if oauthConfig.SSOEnabled && oauthConfig.GoogleClientSecret == "" {
			log.Warn().Msg("SSO is enabled but GOOGLE_OAUTH_CLIENT_SECRET is not set")
		}
		if oauthConfig.SSOEnabled && oauthConfig.RedirectURI == "" {
			log.Warn().Msg("SSO is enabled but GOOGLE_OAUTH_REDIRECT_URI is not set")
		}
	})
	return oauthConfig
}

func getEnvOrViper(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	if viper.IsSet(key) {
		return viper.GetString(key)
	}
	return defaultValue
}

func getEnvOrViperBool(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return defaultValue
}

func getEnvOrViperInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
			return intVal
		}
	}
	if viper.IsSet(key) {
		return viper.GetInt(key)
	}
	return defaultValue
}
