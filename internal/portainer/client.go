package portainer

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client handles communication with the Portainer API
type Client struct {
	baseURL       string
	apiToken      string
	skipTLSVerify bool
	httpClient    *http.Client
}

// NewClient creates a new Portainer API client
func NewClient(baseURL, apiToken string) *Client {
	return NewClientWithTLS(baseURL, apiToken, true) // Default to skip TLS verify
}

// NewClientWithTLS creates a new Portainer API client with TLS verification control
func NewClientWithTLS(baseURL, apiToken string, skipTLSVerify bool) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSVerify,
		},
	}

	return &Client{
		baseURL:       baseURL,
		apiToken:      apiToken,
		skipTLSVerify: skipTLSVerify,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// GetEnvironments retrieves all available environments from Portainer
func (c *Client) GetEnvironments() ([]Environment, error) {
	req, err := c.newRequest("GET", "/api/endpoints", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var environments []Environment
	if err := json.NewDecoder(resp.Body).Decode(&environments); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return environments, nil
}

// GetStack retrieves a stack by name and environment ID
func (c *Client) GetStack(name string, environmentID int) (*Stack, error) {
	// Get all stacks and filter by name and environment
	req, err := c.newRequest("GET", "/api/stacks", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var stacks []Stack
	if err := json.NewDecoder(resp.Body).Decode(&stacks); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Find stack with matching name and environment ID
	for _, stack := range stacks {
		if stack.Name == name && stack.EnvironmentID == environmentID {
			return &stack, nil
		}
	}

	return nil, nil // Stack not found
}

// CreateStack creates a new stack in Portainer
func (c *Client) CreateStack(name, composeContent string, environmentID int) (*Stack, error) {
	// Create JSON request body
	reqBody := map[string]string{
		"name":             name,
		"stackFileContent": composeContent,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use the correct endpoint for Docker Compose stack creation
	endpoint := fmt.Sprintf("/api/stacks/create/standalone/string?endpointId=%d", environmentID)
	req, err := c.newRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleErrorResponse(resp)
	}

	var stack Stack
	if err := json.NewDecoder(resp.Body).Decode(&stack); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stack, nil
}

// UpdateStack updates an existing stack in Portainer
func (c *Client) UpdateStack(stackID int, composeContent string, pullImages bool, environmentID int) error {
	// Create JSON request body for stack update
	reqBody := map[string]interface{}{
		"prune":            true,
		"pullImage":        pullImages,
		"stackFileContent": composeContent,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use the correct endpoint for stack updates with endpointId parameter
	endpoint := fmt.Sprintf("/api/stacks/%d?endpointId=%d", stackID, environmentID)
	req, err := c.newRequest("PUT", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	return nil
}

// GetStackDetails retrieves detailed stack information by ID
func (c *Client) GetStackDetails(stackID int) (*StackDetails, error) {
	endpoint := fmt.Sprintf("/api/stacks/%d", stackID)
	req, err := c.newRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var stackDetails StackDetails
	if err := json.NewDecoder(resp.Body).Decode(&stackDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stackDetails, nil
}

// GetStackContainers retrieves containers for a specific stack via Docker proxy
func (c *Client) GetStackContainers(environmentID int, stackName string) ([]Container, error) {
	// Create filters for Docker Compose project label
	// Docker API expects filters in the format: {"label": ["com.docker.compose.project=stackname"]}
	filters := map[string][]string{
		"label": {fmt.Sprintf("com.docker.compose.project=%s", stackName)},
	}

	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filters: %w", err)
	}

	// URL encode the filters
	encodedFilters := url.QueryEscape(string(filtersJSON))

	// Use Docker proxy endpoint to list containers
	endpoint := fmt.Sprintf("/api/endpoints/%d/docker/containers/json?filters=%s", environmentID, encodedFilters)
	req, err := c.newRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var containers []Container
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return containers, nil
}

// newRequest creates a new HTTP request with proper headers
func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	// Ensure baseURL ends with /
	baseURL := c.baseURL
	if baseURL[len(baseURL)-1:] != "/" {
		baseURL += "/"
	}

	// Remove leading slash from path
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	fullURL := baseURL + path
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", c.apiToken)
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// handleErrorResponse processes error responses from the API
func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("API request failed with status %d: failed to read error response", resp.StatusCode)
	}

	// If response body is empty, return a simple error
	if len(body) == 0 {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if apiErr.Message != "" {
		return fmt.Errorf("API error: %s", apiErr.Message)
	}

	return fmt.Errorf("API request failed with status %d", resp.StatusCode)
}

// ValidateURL checks if the provided URL is valid
func ValidateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must include scheme (http:// or https://)")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include host")
	}

	return nil
}
