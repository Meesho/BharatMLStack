package argocd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)


type ArgoCDConfig struct {
	API   string
	Token string
}

// InitArgoCDClient initializes ArgoCD client configuration
func InitArgoCDClient(configs map[string]ArgoCDConfig) {
	log.Info().Msg("ArgoCD client initialized")
}

// getArgoCDAPI returns the ArgoCD API endpoint for the given workingEnv
// Uses workingEnv exactly as provided (just uppercased) - no transformation
// Falls back to ARGOCD_API if environment-specific value is not set
// Example: For workingEnv="gcp_stg", reads GCP_STG_ARGOCD_API
func getArgoCDAPI(workingEnv string) string {
	// Use workingEnv exactly as provided (just uppercased) - no transformation
	// Example: "gcp_stg" -> "GCP_STG_ARGOCD_API", "gcp_int" -> "GCP_INT_ARGOCD_API"
	envSpecificKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_API")
	api := viper.GetString(envSpecificKey)
	
	if api != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Using environment-specific ArgoCD API")
		return api
	}
	
	// Fallback to default ArgoCD API if environment-specific not set
	api = viper.GetString("ARGOCD_API")
	if api != "" {
		log.Debug().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Environment-specific ArgoCD API not found, using default ARGOCD_API")
	} else {
		log.Warn().
			Str("workingEnv", workingEnv).
			Str("env_key", envSpecificKey).
			Msg("Neither environment-specific nor default ArgoCD API is configured")
	}
	return api
}

func getArgoCDClient(api string, reqBody []byte, method string, workingEnv string, workingBU string, zone string) (*http.Request, error) {
	log.Info().
		Str("API", api).
		Str("Method", method).
		Str("WorkingEnv", workingEnv).
		Str("WorkingBU", workingBU).
		Str("Zone", zone).
		Msg("Entered getArgoCDClient func")

	// Use workingEnv exactly as provided (just uppercased) - no transformation
	// Examples: "gcp_stg" -> "GCP_STG_ARGOCD_API", "stg" -> "STG_ARGOCD_API"
	hostKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_API")
	tokenKey := GetArgoCDConfigKey(workingEnv, "ARGOCD_TOKEN")

	// If zone = ase1c → look for zoneC overrides
	if strings.EqualFold(zone, "ase1c") {
		// Use workingEnv exactly as provided for zone overrides too
		workingEnvUpper := strings.ToUpper(workingEnv)
		zHostKey := workingEnvUpper + "_ARGOCD_ASE1C_API"
		zTokenKey := workingEnvUpper + "_ARGOCD_ASE1C_TOKEN"

		if viper.GetString(zHostKey) != "" {
			log.Info().Str("ZApiKey", zHostKey).Msg("Using ZoneC API override")
			hostKey = zHostKey
		}
		if viper.GetString(zTokenKey) != "" {
			log.Info().Str("ZTokKey", zTokenKey).Msg("Using ZoneC TOKEN override")
			tokenKey = zTokenKey
		}
	}

	var argocdApi string
	if strings.HasPrefix(api, "https://") {
		argocdApi = api
	} else {
		// Try environment-specific first, then fallback to default
		envApi := viper.GetString(hostKey)
		if envApi != "" {
			argocdApi = envApi + api
		} else {
			// Fallback to default ARGOCD_API
			defaultApi := viper.GetString("ARGOCD_API")
			if defaultApi != "" {
				argocdApi = defaultApi + api
			} else {
				return nil, fmt.Errorf("ArgoCD API not configured (checked %s and ARGOCD_API)", hostKey)
			}
		}
	}

	// Try environment-specific first, then fallback to default
	argocdToken := viper.GetString(tokenKey)
	if argocdToken == "" {
		argocdToken = viper.GetString("ARGOCD_TOKEN")
		if argocdToken == "" {
			return nil, fmt.Errorf("ArgoCD Token not configured (checked %s and ARGOCD_TOKEN)", tokenKey)
		}
	}

	req, err := http.NewRequest(method, argocdApi, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Error().Err(err).Msg("getArgoCDClient: Failed to create HTTP request")
		return req, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+argocdToken)

	return req, nil
}

func makeArgoCDCall(req *http.Request) ([]byte, error) {
	log.Info().Msg("Entered makeArgoCDCall")
	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("makeArgoCDCall: Failed to get call response from ArgoCD")
		return nil, errors.New("failed to get call response from ArgoCD")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Msg("makeArgoCDCall: Failed to parse response from ArgoCD")
		return nil, errors.New("failed to parse response from ArgoCD")
	}
	defer response.Body.Close()

	if response.StatusCode == 403 {
		log.Error().Msg("makeArgoCDCall: Got 403 response from ArgoCD application does not exist")
		return body, errors.New("got 403 reponse code from ArgoCD -> application does not exist")
	}

	if response.StatusCode != 200 {
		log.Error().Int("StatusCode", response.StatusCode).Msg("makeArgoCDCall: Received non-200 response from ArgoCD")
		return nil, fmt.Errorf("did not get 200 response from ArgoCD, got %d", response.StatusCode)
	}
	log.Info().Msg("Successfully received response from ArgoCD")
	return body, nil
}

