package argocd

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// Application resource-tree Information
type ArgoCDAppHealthStatus struct {
	Status string `json:"status"`
}

type ArgoCDResourceRev struct {
	Value string `json:"value"`
}

type ArgoCDAppResourcesDetail struct {
	Kind   string                `json:"kind"`
	Name   string                `json:"name"`
	Health ArgoCDAppHealthStatus `json:"health"`
	Info   []ArgoCDResourceRev   `json:"info"`
}

type ArgoCDAppApiResponse struct {
	Nodes []ArgoCDAppResourcesDetail `json:"nodes"`
}

type ArgoCDApplicationWrapper struct {
	Metadata ArgoCDApplicationMetadata `json:"metadata"`
	Status   ArgoCDApplicationStatus   `json:"status"`
}

type ArgoCDApplicationMetadata struct {
	Name   string                  `json:"name"`
	Labels ArgoCDApplicationLabels `json:"labels"`
}

type ArgoCDApplicationStatus struct {
	Sync struct {
		Status   string `json:"status"`
		Revision string `json:"revision"`
	} `json:"sync"`
	Health struct {
		Status string `json:"status"`
	} `json:"health"`
	OperationState struct {
		Phase      string `json:"phase"`
		Message    string `json:"message"`
		StartedAt  string `json:"startedAt"`
		FinishedAt string `json:"finishedAt"`
	}
	Summary struct {
		Images []string `json:"images"`
	}
}

type ArgoCDApplicationLabels struct {
	Env            string `json:"env"`
	Team           string `json:"team"`
	BU             string `json:"bu"`
	Service        string `json:"service"`
	AppName        string `json:"app_name"`
	PriorityV2     string `json:"priority_v2"`
	PrimaryOwner   string `json:"primary_owner"`
	SecondaryOwner string `json:"secondary_owner"`
	CommitId       string `json:"commit_id"`
}

type ArgoCDApplicationListDto struct {
	Items    []ArgoCDApplicationWrapper `json:"items"`
	Metadata interface{}                `json:"metadata"`
}

func GetArgoCDApplication(name string, workingEnv string) (ArgoCDApplicationWrapper, error) {
	log.Info().Str("workingEnv", workingEnv).Str("app name", name).Msg("In GetArgoCDApplication")
	req, err := getArgoCDClient(
		"/api/v1/applications/"+name+"?fields=items.metadata.name,items.metadata.labels",
		[]byte{},
		"GET",
		workingEnv,
		"", // empty workingBU, magically pick
		"",
	)
	if err != nil {
		log.Error().Err(err).Msg("GetArgoCDApplication: Failed while calling getArgoCDClient in GetArgoCDApplication")
		return ArgoCDApplicationWrapper{}, err
	}

	responseBody, err := makeArgoCDCall(req)
	if err != nil {
		log.Error().Err(err).Msg("GetArgoCDApplication: Failed while calling makeArgoCDCall in GetArgoCDApplication")
		return ArgoCDApplicationWrapper{}, err
	}
	var result ArgoCDApplicationWrapper
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		log.Error().Err(err).Msg("GetArgoCDApplication: Failed to Unmarshal ArgoCDApplicationWrapper in GetArgoCDApplication")
		return ArgoCDApplicationWrapper{}, err
	}

	return result, nil
}

func IsCanary(applicationName string, workingEnv string) bool {
	log.Info().Str("workingEnv", workingEnv).Str("service", applicationName).Msg("In IsCanary")
	argocdResource, err := GetArgoCDResource("Canary", applicationName, false)
	if err != nil {
		log.Error().Err(err).Msg("IsCanary: Failed while calling argocdResource")
		return false
	}

	_, err = argocdResource.FetchResourceFromArgoCD(workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("IsCanary: Failed while calling FetchResourceFromArgoCD")
		return false
	}

	return true
}

func GetAllArgoCDApplicationList(workingEnv string) ([]ArgoCDApplicationMetadata, error) {
	log.Info().Str("workingEnv", workingEnv).Msg("In GetAllArgoCDApplicationList")
	// Based on the working env, identify what all BUs to call
	var busToCall []string
	if workingEnv == "gcp_prd" {
		busToCall = []string{"common", "central", "catalog", "supply", "growth", "finance"} // SUPPORTED_GCP_BUS
	} else if workingEnv == "gcp_int" {
		busToCall = []string{"common", "shared"} // SUPPORTED_GCP_BUS_INT
	} else {
		// For now, if not GCP prd/int, means no BU
		busToCall = []string{"common"}
	}

	var applicationList []ArgoCDApplicationMetadata

	var wg sync.WaitGroup
	channel := make(chan []ArgoCDApplicationMetadata)

	for _, v := range busToCall {
		wg.Add(1)
		go intermediateWaitGroupHandler(workingEnv, v, channel, &wg)
	}

	// Waitgroup for reading the channel
	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func(applicationList []ArgoCDApplicationMetadata) {
		for v := range channel {
			applicationList = append(applicationList, v...)
		}
		wg2.Done()
	}(applicationList)

	// Wait for all results to come in
	wg.Wait()

	// Close the channel, once all results are in so that applicationList can be considered as complete
	close(channel)

	// Wait till the channel is closed properly
	wg2.Wait()
	return applicationList, nil
}

func intermediateWaitGroupHandler(workingEnv string, workingBU string, channel chan []ArgoCDApplicationMetadata, wg *sync.WaitGroup) {
	log.Info().Str("workingEnv", workingEnv).Str("workingBU", workingBU).Msg("Entered intermediateWaitGroupHandler function")
	defer wg.Done()
	// Here we manipulate the workingEnv, so that the correct argocd is called
	apps, err := GetArgoCDApplicationList(workingEnv, workingBU)
	if err == nil {
		log.Info().Msg("Got ArgoCDApplicationcList")
		channel <- apps
	}
}

func GetArgoCDApplicationList(workingEnv string, workingBU string) ([]ArgoCDApplicationMetadata, error) {
	log.Info().Str("workingEnv", workingEnv).Msg("In GetArgoCDApplicationList")
	req, err := getArgoCDClient(
		"/applications?fields=items.metadata.name,items.metadata.labels",
		[]byte{},
		"GET",
		workingEnv,
		workingBU,
		"",
	)
	if err != nil {
		log.Error().Err(err).Str("workingEnv", workingEnv).Str("workingBU", workingBU).Msg("GetArgoCDApplicationList: Failed to getArgoCDClient")
		return nil, err
	}

	responseBody, err := makeArgoCDCall(req)
	if err != nil {
		log.Error().Err(err).Str("workingEnv", workingEnv).Str("workingBU", workingBU).Msg("GetArgoCDApplicationList: Failed to makeArgoCDCall")
		return nil, err
	}

	var result ArgoCDApplicationListDto
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		log.Error().Err(err).Str("workingEnv", workingEnv).Str("workingBU", workingBU).Msg("GetArgoCDApplicationList: Failed to Unmarshal json")
		return nil, err
	}

	env := workingEnv
	// Strip gcp_ prefix for other envs
	if strings.HasPrefix(workingEnv, "gcp_") {
		env = strings.Replace(workingEnv, "gcp_", "", 1)
	}

	var applicationList []ArgoCDApplicationMetadata
	for _, item := range result.Items {
		// Skip services without the "service" label as this functionality is for applications and not for tools
		if item.Metadata.Labels.Service == "" {
			continue
		}

		if item.Metadata.Labels.Env == env && strings.HasPrefix(item.Metadata.Name, env+"-") {
			applicationList = append(applicationList, item.Metadata)
		}
	}
	return applicationList, nil
}
