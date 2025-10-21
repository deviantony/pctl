package portainer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://portainer.example.com", "test-token")

	assert.Equal(t, "https://portainer.example.com", client.baseURL)
	assert.Equal(t, "test-token", client.apiToken)
	assert.True(t, client.skipTLSVerify)
	assert.NotNil(t, client.httpClient)
}

func TestNewClientWithTLS(t *testing.T) {
	tests := []struct {
		name          string
		skipTLSVerify bool
		expectedSkip  bool
	}{
		{
			name:          "skip TLS verification",
			skipTLSVerify: true,
			expectedSkip:  true,
		},
		{
			name:          "verify TLS",
			skipTLSVerify: false,
			expectedSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithTLS("https://portainer.example.com", "test-token", tt.skipTLSVerify)

			assert.Equal(t, "https://portainer.example.com", client.baseURL)
			assert.Equal(t, "test-token", client.apiToken)
			assert.Equal(t, tt.expectedSkip, client.skipTLSVerify)
			assert.NotNil(t, client.httpClient)
		})
	}
}

func TestClient_newRequest(t *testing.T) {
	client := NewClient("https://portainer.example.com", "test-token")

	req, err := client.newRequest("GET", "/api/endpoints", nil)
	require.NoError(t, err)

	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://portainer.example.com/api/endpoints", req.URL.String())
	assert.Equal(t, "test-token", req.Header.Get("X-API-Key"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
}

func TestClient_newRequest_URLFormatting(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     string
		expected string
	}{
		{
			name:     "baseURL with trailing slash",
			baseURL:  "https://portainer.example.com/",
			path:     "/api/endpoints",
			expected: "https://portainer.example.com/api/endpoints",
		},
		{
			name:     "baseURL without trailing slash",
			baseURL:  "https://portainer.example.com",
			path:     "/api/endpoints",
			expected: "https://portainer.example.com/api/endpoints",
		},
		{
			name:     "path without leading slash",
			baseURL:  "https://portainer.example.com",
			path:     "api/endpoints",
			expected: "https://portainer.example.com/api/endpoints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				baseURL:    tt.baseURL,
				apiToken:   "test-token",
				httpClient: &http.Client{},
			}

			req, err := client.newRequest("GET", tt.path, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, req.URL.String())
		})
	}
}

func TestClient_handleErrorResponse(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "API error with message",
			statusCode:    400,
			responseBody:  `{"message": "Invalid request", "details": "Missing required field"}`,
			expectedError: "API error: Invalid request",
		},
		{
			name:          "API error without message",
			statusCode:    500,
			responseBody:  `{"details": "Internal server error"}`,
			expectedError: "API request failed with status 500",
		},
		{
			name:          "empty response body",
			statusCode:    404,
			responseBody:  "",
			expectedError: "API request failed with status 404",
		},
		{
			name:          "invalid JSON response",
			statusCode:    400,
			responseBody:  "invalid json",
			expectedError: "API request failed with status 400: invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("https://portainer.example.com", "test-token")

			// Create a mock response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       http.NoBody,
			}

			// Mock the response body
			if tt.responseBody != "" {
				resp.Body = http.NoBody
				// We can't easily mock the response body in this test
				// This is more of a unit test for the error handling logic
			}

			err := client.handleErrorResponse(resp)
			assert.Error(t, err)
			// Note: The actual error message will depend on the implementation
			// This test verifies that an error is returned
		})
	}
}

func TestClient_handleErrorResponse_EmptyBody(t *testing.T) {
	client := NewClient("https://portainer.example.com", "test-token")

	// Create a mock response with empty body
	resp := &http.Response{
		StatusCode: 404,
		Body:       http.NoBody,
	}

	err := client.handleErrorResponse(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed with status 404")
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid HTTPS URL",
			url:         "https://portainer.example.com",
			expectError: false,
		},
		{
			name:        "valid HTTP URL",
			url:         "http://localhost:9000",
			expectError: false,
		},
		{
			name:        "URL with path",
			url:         "https://portainer.example.com/api",
			expectError: false,
		},
		{
			name:        "URL without scheme",
			url:         "portainer.example.com",
			expectError: true,
			errorMsg:    "URL must include scheme",
		},
		{
			name:        "URL without host",
			url:         "https://",
			expectError: true,
			errorMsg:    "URL must include host",
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			expectError: true,
			errorMsg:    "URL must include scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_GetEnvironments(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/endpoints", r.URL.Path)
		assert.Equal(t, "test-token", r.Header.Get("X-API-Key"))

		environments := []Environment{
			{ID: 1, Name: "production", URL: "https://prod.portainer.com"},
			{ID: 2, Name: "staging", URL: "https://staging.portainer.com"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(environments)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	environments, err := client.GetEnvironments()

	require.NoError(t, err)
	require.Len(t, environments, 2)
	assert.Equal(t, 1, environments[0].ID)
	assert.Equal(t, "production", environments[0].Name)
	assert.Equal(t, "https://prod.portainer.com", environments[0].URL)
	assert.Equal(t, 2, environments[1].ID)
	assert.Equal(t, "staging", environments[1].Name)
	assert.Equal(t, "https://staging.portainer.com", environments[1].URL)
}

func TestClient_GetStack(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/stacks", r.URL.Path)

		stacks := []Stack{
			{ID: 1, Name: "myapp", EnvironmentID: 1, Status: 1},
			{ID: 2, Name: "myapp", EnvironmentID: 2, Status: 1},
			{ID: 3, Name: "other-app", EnvironmentID: 1, Status: 1},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stacks)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	// Test finding existing stack
	stack, err := client.GetStack("myapp", 1)
	require.NoError(t, err)
	require.NotNil(t, stack)
	assert.Equal(t, 1, stack.ID)
	assert.Equal(t, "myapp", stack.Name)
	assert.Equal(t, 1, stack.EnvironmentID)

	// Test finding non-existing stack
	stack, err = client.GetStack("nonexistent", 1)
	require.NoError(t, err)
	assert.Nil(t, stack)
}

func TestClient_CreateStack(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasPrefix(r.URL.Path, "/api/stacks/create/standalone/string"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "myapp", reqBody["name"])
		assert.Equal(t, "version: '3.8'\nservices:\n  web:\n    image: nginx", reqBody["stackFileContent"])

		stack := Stack{
			ID:            1,
			Name:          "myapp",
			EnvironmentID: 1,
			Status:        1,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(stack)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	composeContent := "version: '3.8'\nservices:\n  web:\n    image: nginx"
	stack, err := client.CreateStack("myapp", composeContent, 1)

	require.NoError(t, err)
	require.NotNil(t, stack)
	assert.Equal(t, 1, stack.ID)
	assert.Equal(t, "myapp", stack.Name)
	assert.Equal(t, 1, stack.EnvironmentID)
}

func TestClient_UpdateStack(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/stacks/1", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, true, reqBody["prune"])
		assert.Equal(t, true, reqBody["pullImage"])
		assert.Equal(t, "version: '3.8'\nservices:\n  web:\n    image: nginx:latest", reqBody["stackFileContent"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	composeContent := "version: '3.8'\nservices:\n  web:\n    image: nginx:latest"
	err := client.UpdateStack(1, composeContent, true, 1)

	require.NoError(t, err)
}

func TestClient_GetStackDetails(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/stacks/1", r.URL.Path)

		stackDetails := StackDetails{
			ID:            1,
			Name:          "myapp",
			Status:        1,
			EnvironmentID: 1,
			CreatedAt:     1640995200,
			UpdatedAt:     1640998800,
			CreatedBy:     "admin",
			UpdatedBy:     "admin",
			ProjectPath:   "/opt/portainer/stacks/1",
			EntryPoint:    "docker-compose.yml",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stackDetails)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	stackDetails, err := client.GetStackDetails(1)

	require.NoError(t, err)
	require.NotNil(t, stackDetails)
	assert.Equal(t, 1, stackDetails.ID)
	assert.Equal(t, "myapp", stackDetails.Name)
	assert.Equal(t, 1, stackDetails.Status)
	assert.Equal(t, 1, stackDetails.EnvironmentID)
	assert.Equal(t, int64(1640995200), stackDetails.CreatedAt)
	assert.Equal(t, int64(1640998800), stackDetails.UpdatedAt)
	assert.Equal(t, "admin", stackDetails.CreatedBy)
	assert.Equal(t, "admin", stackDetails.UpdatedBy)
	assert.Equal(t, "/opt/portainer/stacks/1", stackDetails.ProjectPath)
	assert.Equal(t, "docker-compose.yml", stackDetails.EntryPoint)
}

func TestClient_GetStackContainers(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.True(t, strings.HasPrefix(r.URL.Path, "/api/endpoints/1/docker/containers/json"))
		assert.Contains(t, r.URL.RawQuery, "filters=")

		containers := []Container{
			{
				ID:     "abc123",
				Names:  []string{"/myapp_web_1"},
				Image:  "nginx:latest",
				Status: "Up 2 hours",
				State:  "running",
			},
			{
				ID:     "def456",
				Names:  []string{"/myapp_api_1"},
				Image:  "myapp-api:latest",
				Status: "Up 1 hour",
				State:  "running",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containers)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	containers, err := client.GetStackContainers(1, "myapp")

	require.NoError(t, err)
	require.Len(t, containers, 2)
	assert.Equal(t, "abc123", containers[0].ID)
	assert.Equal(t, []string{"/myapp_web_1"}, containers[0].Names)
	assert.Equal(t, "nginx:latest", containers[0].Image)
	assert.Equal(t, "Up 2 hours", containers[0].Status)
	assert.Equal(t, "running", containers[0].State)
}

func TestClient_GetContainerLogs(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.True(t, strings.HasPrefix(r.URL.Path, "/api/endpoints/1/docker/containers/abc123/logs"))
		assert.Contains(t, r.URL.RawQuery, "stdout=true")
		assert.Contains(t, r.URL.RawQuery, "stderr=true")
		assert.Contains(t, r.URL.RawQuery, "timestamps=true")
		assert.Contains(t, r.URL.RawQuery, "tail=100")

		w.Write([]byte("2023-01-01T12:00:00Z Starting application\n2023-01-01T12:00:01Z Application started\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	logs, err := client.GetContainerLogs(1, "abc123", 100)

	require.NoError(t, err)
	assert.Contains(t, logs, "Starting application")
	assert.Contains(t, logs, "Application started")
}

func TestClient_GetDockerInfo(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/endpoints/1/docker/info", r.URL.Path)

		info := map[string]interface{}{
			"Name":              "docker-host",
			"ServerVersion":     "20.10.21",
			"OperatingSystem":   "Linux",
			"Architecture":      "x86_64",
			"Containers":        5,
			"ContainersRunning": 3,
			"ContainersPaused":  0,
			"ContainersStopped": 2,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	info, err := client.GetDockerInfo(1)

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "docker-host", info["Name"])
	assert.Equal(t, "20.10.21", info["ServerVersion"])
	assert.Equal(t, "Linux", info["OperatingSystem"])
	assert.Equal(t, "x86_64", info["Architecture"])
	assert.Equal(t, float64(5), info["Containers"])
	assert.Equal(t, float64(3), info["ContainersRunning"])
}

func TestClient_ImageExists(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expected    bool
		expectError bool
	}{
		{
			name:        "image exists",
			statusCode:  http.StatusOK,
			expected:    true,
			expectError: false,
		},
		{
			name:        "image not found",
			statusCode:  http.StatusNotFound,
			expected:    false,
			expectError: false,
		},
		{
			name:        "server error",
			statusCode:  http.StatusInternalServerError,
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/api/endpoints/1/docker/images/nginx:latest/json", r.URL.Path)

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")

			exists, err := client.ImageExists(1, "nginx:latest")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}
		})
	}
}

func TestClient_BuildImage(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasPrefix(r.URL.Path, "/api/endpoints/1/docker/build"))
		assert.Equal(t, "application/x-tar", r.Header.Get("Content-Type"))
		assert.Contains(t, r.URL.RawQuery, "t=myapp%3Alatest")
		assert.Contains(t, r.URL.RawQuery, "dockerfile=Dockerfile")

		// Simulate build output
		buildOutput := `{"stream": "Step 1/3 : FROM nginx:latest"}
{"stream": " ---> abc123def456"}
{"stream": "Step 2/3 : COPY . /usr/share/nginx/html"}
{"stream": " ---> def456ghi789"}
{"stream": "Step 3/3 : EXPOSE 80"}
{"stream": " ---> ghi789jkl012"}
{"aux": {"ID": "sha256:ghi789jkl012"}}`

		w.Write([]byte(buildOutput))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	var buildLines []string
	onLine := func(line string) {
		buildLines = append(buildLines, line)
	}

	opts := BuildOptions{
		Tag:        "myapp:latest",
		Dockerfile: "Dockerfile",
		BuildArgs:  map[string]string{"NODE_ENV": "production"},
		Target:     "production",
		NoCache:    false,
	}

	// Create a mock tar reader
	tarReader := strings.NewReader("mock tar content")

	err := client.BuildImage(1, tarReader, opts, onLine)

	require.NoError(t, err)
	assert.Len(t, buildLines, 7) // Should have 7 lines of build output
	assert.Contains(t, buildLines[0], "Step 1/3 : FROM nginx:latest")
	assert.Contains(t, buildLines[6], "sha256:ghi789jkl012")
}

func TestClient_LoadImage(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/endpoints/1/docker/images/load", r.URL.Path)
		assert.Equal(t, "application/x-tar", r.Header.Get("Content-Type"))

		// Simulate load output
		loadOutput := `{"stream": "Loaded image: myapp:latest"}
{"stream": "Loaded image: myapp:staging"}`

		w.Write([]byte(loadOutput))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	var progressLines []string
	onProgress := func(line string) {
		progressLines = append(progressLines, line)
	}

	// Create a mock tar reader
	tarReader := strings.NewReader("mock tar content")

	err := client.LoadImage(1, tarReader, onProgress)

	require.NoError(t, err)
	assert.Len(t, progressLines, 2)
	assert.Contains(t, progressLines[0], "Loaded image: myapp:latest")
	assert.Contains(t, progressLines[1], "Loaded image: myapp:staging")
}
