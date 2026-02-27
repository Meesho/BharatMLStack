package argocd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rs/zerolog/log"
)

// ApplicationLogsOptions holds optional parameters for GetApplicationLogs.
// Streaming (follow) is not supported; reject follow=true at the call site.
// Defaults: Previous=false, SinceSeconds=0, Filter="".
type ApplicationLogsOptions struct {
	PodName      string
	Container    string
	Previous     bool
	SinceSeconds int64
	TailLines    int64
	Filter       string
}

// ApplicationLogResult is the inner result object in a log entry.
type ApplicationLogResult struct {
	Content      string  `json:"content"`
	TimeStamp    *string `json:"timeStamp"`
	Last         bool    `json:"last"`
	TimeStampStr string  `json:"timeStampStr"`
	PodName      string  `json:"podName"`
}

// ApplicationLogEntry is a single log entry (wrapper with "result").
type ApplicationLogEntry struct {
	Result ApplicationLogResult `json:"result"`
}

// ApplicationLogsErrorDetails is the error body returned by Argo CD on non-2xx.
type ApplicationLogsErrorDetails struct {
	Details    []interface{} `json:"details"`
	GrpcCode   int           `json:"grpc_code"`
	HttpCode   int           `json:"http_code"`
	HttpStatus string        `json:"http_status"`
	Message    string        `json:"message"`
}

// ArgoCDAPIError is a structured error returned when the Argo CD API
// responds with a non-2xx status code. Callers can use errors.As to
// extract the upstream HTTP status and forward it appropriately.
type ArgoCDAPIError struct {
	StatusCode int
	HTTPStatus string
	Message    string
}

func (e *ArgoCDAPIError) Error() string {
	return fmt.Sprintf("argo CD logs API error (HTTP %d - %s): %s",
		e.StatusCode, e.HTTPStatus, e.Message)
}

func validateLogsOptions(opts *ApplicationLogsOptions) error {
	if opts == nil {
		return fmt.Errorf("log options must not be nil")
	}
	if opts.Container == "" {
		return fmt.Errorf("container is required")
	}
	return nil
}

func buildLogsURL(workingEnv, name string, opts *ApplicationLogsOptions) (string, error) {
	base := getArgoCDAPI(workingEnv)
	if base == "" {
		return "", fmt.Errorf("ArgoCD API not configured for environment %s", workingEnv)
	}
	path := "/api/v1/applications/" + url.PathEscape(name) + "/logs"
	u, err := url.Parse(base + path)
	if err != nil {
		return "", fmt.Errorf("failed to parse logs URL: %w", err)
	}
	q := url.Values{}
	q.Set("container", opts.Container)
	if opts.Previous {
		q.Set("previous", "true")
	}
	if opts.SinceSeconds > 0 {
		q.Set("sinceSeconds", strconv.FormatInt(opts.SinceSeconds, 10))
	}
	tailLines := int64(1000)
	if opts.TailLines > 0 {
		tailLines = opts.TailLines
	}
	q.Set("tailLines", strconv.FormatInt(tailLines, 10))
	if opts.Filter != "" {
		q.Set("filter", opts.Filter)
	}
	if opts.PodName != "" {
		q.Set("podName", opts.PodName)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// parseNDJSONLogs decodes NDJSON from r (e.g. resp.Body) without buffering the full stream.
func parseNDJSONLogs(r io.Reader) ([]ApplicationLogEntry, error) {
	entries := make([]ApplicationLogEntry, 0)
	dec := json.NewDecoder(r)
	for {
		var e ApplicationLogEntry
		err := dec.Decode(&e)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode log entry: %w", err)
		}
		if e.Result.Last {
			break
		}
		if e.Result.TimeStamp == nil {
			return nil, fmt.Errorf("argo log stream error: %s", e.Result.Content)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// GetApplicationLogs fetches application logs from Argo CD (non-streaming only).
// name is the Argo CD application name. opts cannot be nil
// When follow=true, returns an error that streaming is not supported.
func GetApplicationLogs(name, workingEnv string, opts *ApplicationLogsOptions) ([]ApplicationLogEntry, error) {
	if name == "" {
		return nil, fmt.Errorf("application name is required")
	}
	if err := validateLogsOptions(opts); err != nil {
		return nil, err
	}

	fullURL, err := buildLogsURL(workingEnv, name, opts)
	if err != nil {
		log.Error().Err(err).Str("name", name).Str("workingEnv", workingEnv).Msg("GetApplicationLogs: failed to build URL")
		return nil, err
	}

	req, err := getArgoCDClient(fullURL, nil, http.MethodGet, workingEnv, "", "")
	if err != nil {
		log.Error().Err(err).Str("name", name).Str("workingEnv", workingEnv).Msg("GetApplicationLogs: failed to create request")
		return nil, err
	}

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("name", name).Str("workingEnv", workingEnv).Msg("GetApplicationLogs: request failed")
		return nil, fmt.Errorf("failed to get logs from Argo CD: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var wrapper struct {
			Error ApplicationLogsErrorDetails `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
			return nil, &ArgoCDAPIError{
				StatusCode: resp.StatusCode,
				HTTPStatus: http.StatusText(resp.StatusCode),
				Message:    fmt.Sprintf("failed to decode error response: %v", err),
			}
		}
		msg := wrapper.Error.Message
		if msg == "" {
			msg = "no error message in response"
		}
		httpStatus := wrapper.Error.HttpStatus
		if httpStatus == "" {
			httpStatus = http.StatusText(resp.StatusCode)
		}
		log.Debug().
			Str("name", name).
			Int("status", resp.StatusCode).
			Str("message", msg).
			Msg("GetApplicationLogs: Argo CD returned error")
		return nil, &ArgoCDAPIError{
			StatusCode: resp.StatusCode,
			HTTPStatus: httpStatus,
			Message:    msg,
		}
	}

	entries, err := parseNDJSONLogs(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("name", name).Msg("GetApplicationLogs: failed to parse response")
		return nil, err
	}
	log.Info().Str("name", name).Str("workingEnv", workingEnv).Int("count", len(entries)).Int("status", resp.StatusCode).Msg("GetApplicationLogs: success")
	return entries, nil
}
