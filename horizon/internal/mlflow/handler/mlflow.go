package handler

import (
	"io"
	"net/http"
	"strings"
	"time"

	mlflowpkg "github.com/Meesho/BharatMLStack/horizon/internal/mlflow"
	"github.com/rs/zerolog/log"
)

type MLFlowHandler struct {
	httpClient *http.Client
	baseURL    string
}

func NewMLFlowHandler() *MLFlowHandler {
	return &MLFlowHandler{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: mlflowpkg.MLFlowHostURL,
	}
}

// ConstructMLFlowURL builds the full MLflow URL from the base URL and path.
func (h *MLFlowHandler) ConstructMLFlowURL(mlflowPath string) string {
	return h.baseURL + mlflowPath
}

// ForwardRequest forwards the HTTP request to the MLFlow backend
func (h *MLFlowHandler) ForwardRequest(method, url string, body io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to create MLFlow request")
		return nil, err
	}

	// RFC 2616 §13.5.1 hop-by-hop headers and proxy-specific headers; never forwarded
	skipHeaders := map[string]bool{
		http.CanonicalHeaderKey("Connection"):          true,
		http.CanonicalHeaderKey("Keep-Alive"):          true,
		http.CanonicalHeaderKey("Proxy-Authenticate"):  true,
		http.CanonicalHeaderKey("Proxy-Authorization"): true,
		http.CanonicalHeaderKey("TE"):                  true,
		http.CanonicalHeaderKey("Trailer"):            true,
		http.CanonicalHeaderKey("Transfer-Encoding"): true,
		http.CanonicalHeaderKey("Upgrade"):            true,
		http.CanonicalHeaderKey("Origin"):             true,
		http.CanonicalHeaderKey("Referer"):            true,
		http.CanonicalHeaderKey("Host"):               true,
	}

	// Connection header lists additional hop-by-hop header names to not forward
	if conn := headers.Get("Connection"); conn != "" {
		for _, token := range strings.Split(conn, ",") {
			name := strings.TrimSpace(token)
			if name != "" {
				skipHeaders[http.CanonicalHeaderKey(name)] = true
			}
		}
	}

	for key, values := range headers {
		if skipHeaders[http.CanonicalHeaderKey(key)] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make the request to MLFlow backend
	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to forward request to MLFlow")
		return nil, err
	}

	return resp, nil
}

// GetHTTPClient returns the HTTP client instance
func (h *MLFlowHandler) GetHTTPClient() *http.Client {
	return h.httpClient
}
