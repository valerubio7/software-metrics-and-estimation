package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIResponse preserves the API status and body, including validation details.
type APIResponse struct {
	StatusCode int
	Body       []byte
}

// apiClient is the JSON-over-HTTP helper shared by the resource clients.
type apiClient struct {
	baseURL string
	http    *http.Client
}

func newAPIClient(baseURL string) apiClient {
	return apiClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// do sends the request, serializing input as JSON when it is not nil.
// It returns an error only for serialization or transport failures; any HTTP
// status is returned as an APIResponse with its body intact.
func (c apiClient) do(ctx context.Context, method, path string, input any) (APIResponse, error) {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return APIResponse{}, err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return APIResponse{}, err
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return APIResponse{}, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return APIResponse{}, err
	}
	return APIResponse{StatusCode: response.StatusCode, Body: responseBody}, nil
}

// ProbeAPI checks API availability without mutating data.
// Any HTTP response counts as available; only network errors or timeouts report unavailability.
func ProbeAPI(baseURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(baseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
