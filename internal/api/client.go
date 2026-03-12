package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const DefaultBaseURL = "https://api.unified.to"

// Client is an HTTP client for the Unified.to API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new API client. If baseURL is empty, uses the default.
func NewClient(apiKey string, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

// Do performs an HTTP request against the Unified.to API.
// Returns the response body, status code, and any error.
func (c *Client) Do(method, category, connectionID, object, id string, body []byte, params map[string]string) ([]byte, int, error) {
	// Build URL path
	path := fmt.Sprintf("/%s/%s/%s", category, connectionID, object)
	if id != "" {
		path = fmt.Sprintf("%s/%s", path, id)
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Build request
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
