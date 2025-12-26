package argocd

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

type SyncOptions struct {
	Items []string `json:"items"`
}

type ArgoCDObject struct {
	AppNamespace string           `json:"appNamespace"`
	AppName      string           `json:"appName"`
	DryRyn       bool             `json:"dryRun"`
	Revision     string           `json:"revision"`
	Prune        bool             `json:"prune"`
	Resources    []ArgoCDResource `json:"resources"`
	SyncOptions  SyncOptions      `json:"syncOptions"`
}

func GetArgoCDObject(applicationName string, resources []ArgoCDResource, workingEnv string, workingBU string) ArgoCDObject {
	log.Info().Str("AppName", applicationName).Str("WorkingEnv", workingEnv).Str("WorkingBU", workingBU).Msg("Entered GetArgoCDObject")
	return ArgoCDObject{
		AppNamespace: GetArgocdNamespace(workingEnv, workingBU),
		AppName:      applicationName,
		DryRyn:       false,
		Revision:     "HEAD", // Default to HEAD, can be configured
		Prune:        false,
		Resources:    resources,
		SyncOptions:  SyncOptions{Items: []string{"CreateNamespace=true"}},
	}
}

func (a ArgoCDObject) Refresh(workingEnv string, workingBU string, zone string) error {
	log.Info().Str("AppName", a.AppName).Str("WorkingEnv", workingEnv).Str("WorkingBU", workingBU).Msg("Entered Refresh function")
	req, err := getArgoCDClient(
		"/applications/"+a.AppName+"?refresh=true",
		nil,
		"GET",
		workingEnv,
		workingBU,
		zone,
	)
	if err != nil {
		log.Error().Err(err).Msg("Refresh: Failed to get ArgoCD client")
		return err
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Refresh: Failed to get refresh call response from ArgoCD")
		return errors.New("failed to get refresh call response from ArgoCD: " + err.Error())
	}
	defer response.Body.Close()
	log.Info().Str("AppName", a.AppName).Msg("Successfully completed Refresh for AppName")
	return nil
}

func (a ArgoCDObject) Sync(workingEnv string, workingBU string, zone string) (map[string]interface{}, error) {
	log.Info().Str("AppName", a.AppName).Str("WorkingEnv", workingEnv).Str("WorkingBU", workingBU).Msg("Entered Sync func")
	if zone == "ase1c" {
		a.AppNamespace = "argocd-" + workingBU + "-ase1c-prd"
	}
	reqBody, err := json.Marshal(a)
	if err != nil {
		log.Error().Err(err).Msg("Sync: Unable to parse request body for argocd resource sync call")
		return nil, errors.New("unable to parse request body for argocd resource sync call")
	}
	req, err := getArgoCDClient(
		"/applications/"+a.AppName+"/sync",
		reqBody,
		"POST",
		workingEnv,
		workingBU,
		zone,
	)
	if err != nil {
		log.Error().Err(err).Msg("Sync: Failed to get ArgoCD client")
		return nil, err
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Sync: Failed to get sync call response from ArgoCD")
		return nil, errors.New("failed to get sync call response from ArgoCD: " + err.Error())
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Msg("Sync: Failed to read response body")
		return nil, errors.New("failed to read response body: " + err.Error())
	}
	var data map[string]interface{}
	err = json.Unmarshal(responseBody, &data)
	if err != nil {
		log.Error().Err(err).Str("Response", string(responseBody)).Msg("Sync: Failed to parse JSON response")
		return nil, errors.New("failed to parse JSON: " + err.Error() + ", response: " + string(responseBody))
	}
	log.Info().Str("AppName", a.AppName).Str("WorkingEnv", workingEnv).Str("Zone", zone).Msg("Successfully completed Sync for AppName")

	return data, nil
}
