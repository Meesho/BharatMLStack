package argocd

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

type ArgoCDResource struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

var groups = map[string][]string{
	"ScaledObject":            {"keda.sh", "v1alpha1"},
	"HorizontalPodAutoscaler": {"autoscaling", "v2"},
	"Canary":                  {"flagger.app", "v1beta1"},
	"Deployment":              {"apps", "v1"},
	"Application":             {"argoproj.io", "v1alpha1"},
	"ReplicaSet":              {"apps", "v1"},
}

func GetArgoCDResource(kind string, name string, hasCanary bool) (ArgoCDResource, error) {
	log.Info().Str("kind", kind).Str("name", name).Bool("hasCanary", hasCanary).Msg("Entered GetArgoCDResource function")
	allowedKinds := []string{"ScaledObject", "HorizontalPodAutoscaler", "Canary", "Deployment", "Application", "ReplicaSet"}

	if !slices.Contains(allowedKinds, kind) {
		log.Error().Msg("GetArgoCDResource: Invalid kind of Object")
		return ArgoCDResource{}, errors.New("Invalid kind of Object")
	}

	group := groups[kind][0]
	version := groups[kind][1]

	fullName := name

	if hasCanary {
		fullName = name + "-primary"
	}
	if kind == "HorizontalPodAutoscaler" {
		fullName = "keda-hpa-" + fullName
	}
	log.Info().Str("group", group).Str("version", version).Str("fullName", fullName).Msg("GetArgoCDResource function")

	return ArgoCDResource{group, kind, fullName, name, version}, nil
}

func (r ArgoCDResource) FetchResourceFromArgoCD(workingEnv string) (interface{}, error) {
	log.Info().Str("workingEnv", workingEnv).Msg("Entered FetchResourceFromArgoCD function")

	baseURL, _ := url.Parse(getArgoCDAPI(workingEnv) + "/applications/" + r.Namespace + "/resource")
	params := url.Values{}
	params.Add("name", r.Namespace)
	params.Add("namespace", r.Namespace)
	params.Add("resourceName", r.Name)
	params.Add("group", r.Group)
	params.Add("kind", r.Kind)
	params.Add("version", r.Version)
	baseURL.RawQuery = params.Encode()

	req, err := getArgoCDClient(
		baseURL.String(),
		[]byte{},
		"GET",
		workingEnv,
		"",
		"",
	)
	if err != nil {
		log.Error().Err(err).Msg("FetchResourceFromArgoCD: cannot create argocd client")
		return nil, err
	}

	responseBody, err := makeArgoCDCall(req)
	if err != nil {
		log.Error().Err(err).Msg("FetchResourceFromArgoCD: cannot make argocd call")
		return nil, err
	}

	var result interface{}
	err = json.Unmarshal(responseBody, &result)

	if err != nil {
		log.Error().Err(err).Msg("FetchResourceFromArgoCD: cannot unmarshal the json")
		return nil, err
	}
	log.Info().Any("result", result).Msg("FetchResourceFromArgoCD function executed successfully")

	return result, nil
}

func (r ArgoCDResource) PatchArgoCDResource(payload interface{}, workingEnv string) (interface{}, error) {
	log.Info().Any("payload", payload).Str("workingEnv", workingEnv).Msg("Entered PatchArgoCDResource function")
	reqBody, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("PatchArgoCDResource: Failed to unmarshal payload")
		return nil, err
	}

	baseURL, _ := url.Parse(getArgoCDAPI(workingEnv) + "/applications/" + r.Namespace + "/resource")
	params := url.Values{}
	params.Add("name", r.Namespace)
	params.Add("namespace", r.Namespace)
	params.Add("resourceName", r.Name)
	params.Add("group", r.Group)
	params.Add("kind", r.Kind)
	params.Add("version", r.Version)
	params.Add("patchType", "application/merge-patch+json")
	baseURL.RawQuery = params.Encode()
	reqBodyString := string(reqBody)
	req, err := getArgoCDClient(
		baseURL.String(),
		[]byte(`"`+strings.ReplaceAll(reqBodyString, `"`, `\"`)+`"`),
		"POST",
		workingEnv,
		"",
		"",
	)
	if err != nil {
		log.Error().Err(err).Msg("PatchArgoCDResource: cannot create ArgoCD client request")
		return nil, err
	}
	if req == nil {
		log.Error().Msg("PatchArgoCDResource: ArgoCD client request is nil")
		return nil, errors.New("ArgoCD client request is nil")
	}

	responseBody, err := makeArgoCDCall(req)
	if err != nil {
		log.Error().Err(err).Msg("PatchArgoCDResource: cannot make makeArgoCDCall")
		return nil, err
	}

	var result interface{}
	err = json.Unmarshal(responseBody, &result)

	if err != nil {
		log.Error().Err(err).Msg("PatchArgoCDResource: cannot unmarshal response")
		return nil, err
	}

	return result, nil
}

func GetArgoCDResourceDetail(appName string, workingEnv string) (ArgoCDAppApiResponse, error) {
	// Construct expected ArgoCD application name
	expectedArgoCDAppName := GetArgocdApplicationNameFromEnv(appName, workingEnv)

	log.Info().
		Str("applicationName", appName).
		Str("environment", workingEnv).
		Str("expectedArgoCDAppName", expectedArgoCDAppName).
		Str("apiPath", "/applications/"+expectedArgoCDAppName+"/resource-tree").
		Msg("GetArgoCDResourceDetail: Starting ArgoCD resource detail lookup")

	appReq, err := getArgoCDClient(
		"/applications/"+expectedArgoCDAppName+"/resource-tree",
		[]byte{},
		"GET",
		workingEnv,
		"",
		"",
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("applicationName", appName).
			Str("workingEnv", workingEnv).
			Str("expectedArgoCDAppName", expectedArgoCDAppName).
			Msg("GetArgoCDResourceDetail: Failed to create ArgoCD client request - check ArgoCD configuration")
		return ArgoCDAppApiResponse{}, err
	}

	log.Info().
		Str("applicationName", appName).
		Str("workingEnv", workingEnv).
		Str("expectedArgoCDAppName", expectedArgoCDAppName).
		Msg("GetArgoCDResourceDetail: ArgoCD client request created, making API call")

	appResBody, err := makeArgoCDCall(appReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("applicationName", appName).
			Str("workingEnv", workingEnv).
			Str("expectedArgoCDAppName", expectedArgoCDAppName).
			Msg("GetArgoCDResourceDetail: Failed to make ArgoCD API call - check ArgoCD connectivity and credentials")
		return ArgoCDAppApiResponse{}, err
	}

	log.Info().
		Str("applicationName", appName).
		Str("workingEnv", workingEnv).
		Str("expectedArgoCDAppName", expectedArgoCDAppName).
		Int("responseBodyLength", len(appResBody)).
		Msg("GetArgoCDResourceDetail: Successfully received response from ArgoCD")

	var appResponse ArgoCDAppApiResponse
	responseBodyStr := string(appResBody)
	if err := json.Unmarshal(appResBody, &appResponse); err != nil {
		responseBodyPreview := responseBodyStr
		if len(responseBodyStr) > 200 {
			responseBodyPreview = responseBodyStr[:200] + "..."
		}
		log.Error().
			Err(err).
			Str("applicationName", appName).
			Str("workingEnv", workingEnv).
			Str("expectedArgoCDAppName", expectedArgoCDAppName).
			Str("responseBodyPreview", responseBodyPreview).
			Msg("GetArgoCDResourceDetail: Failed to unmarshal ArgoCD response body")
		return ArgoCDAppApiResponse{}, err
	}

	log.Info().
		Str("applicationName", appName).
		Str("workingEnv", workingEnv).
		Str("expectedArgoCDAppName", expectedArgoCDAppName).
		Int("nodeCount", len(appResponse.Nodes)).
		Msg("GetArgoCDResourceDetail: Successfully parsed ArgoCD response")

	return appResponse, nil
}

func RestartDeployment(appName string, workingEnv string, isCanary bool) error {
	log.Info().Str("applicationName", appName).Str("environment", workingEnv).Bool("isCanary", isCanary).Msg("Restarting deployment")

	deployment, err := GetArgoCDResource("Deployment", appName, isCanary)
	if err != nil {
		log.Error().Err(err).Msg("RestartDeployment: Failed to get ArgoCD resource")
		return err
	}

	// Generate timestamp for restart annotation
	restartTime := time.Now().UTC().Format(time.RFC3339)

	// Patch the deployment with restart annotation to trigger rolling restart
	_, err = deployment.PatchArgoCDResource(map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/restartedAt": restartTime,
					},
				},
			},
		},
	}, workingEnv)

	if err != nil {
		log.Error().Err(err).Msg("RestartDeployment: Failed to patch deployment for restart")
		return err
	}

	log.Info().
		Str("applicationName", appName).
		Str("environment", workingEnv).
		Bool("isCanary", isCanary).
		Str("restartedAt", restartTime).
		Msg("Successfully triggered deployment restart")

	return nil
}
