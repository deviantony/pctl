package portainer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment_JSONMarshal(t *testing.T) {
	env := Environment{
		ID:   1,
		Name: "production",
		URL:  "https://portainer.example.com",
	}

	jsonData, err := json.Marshal(env)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["Id"])
	assert.Equal(t, "production", result["Name"])
	assert.Equal(t, "https://portainer.example.com", result["URL"])
}

func TestEnvironment_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"Id": 2,
		"Name": "staging",
		"URL": "https://staging.portainer.example.com"
	}`

	var env Environment
	err := json.Unmarshal([]byte(jsonData), &env)
	require.NoError(t, err)

	assert.Equal(t, 2, env.ID)
	assert.Equal(t, "staging", env.Name)
	assert.Equal(t, "https://staging.portainer.example.com", env.URL)
}

func TestStack_JSONMarshal(t *testing.T) {
	stack := Stack{
		ID:            1,
		Name:          "myapp",
		StackFile:     "version: '3.8'\nservices:\n  web:\n    image: nginx",
		EnvironmentID: 1,
		Status:        1,
	}

	jsonData, err := json.Marshal(stack)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["Id"])
	assert.Equal(t, "myapp", result["Name"])
	assert.Equal(t, "version: '3.8'\nservices:\n  web:\n    image: nginx", result["EntryPoint"])
	assert.Equal(t, float64(1), result["EndpointId"])
	assert.Equal(t, float64(1), result["Status"])
}

func TestStack_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"Id": 2,
		"Name": "myapp-staging",
		"EntryPoint": "version: '3.8'\nservices:\n  api:\n    image: myapp-api:latest",
		"EndpointId": 2,
		"Status": 2
	}`

	var stack Stack
	err := json.Unmarshal([]byte(jsonData), &stack)
	require.NoError(t, err)

	assert.Equal(t, 2, stack.ID)
	assert.Equal(t, "myapp-staging", stack.Name)
	assert.Equal(t, "version: '3.8'\nservices:\n  api:\n    image: myapp-api:latest", stack.StackFile)
	assert.Equal(t, 2, stack.EnvironmentID)
	assert.Equal(t, 2, stack.Status)
}

func TestContainer_JSONMarshal(t *testing.T) {
	container := Container{
		ID:      "abc123def456",
		Names:   []string{"/myapp_web_1", "/myapp_web"},
		Image:   "nginx:latest",
		Status:  "Up 2 hours",
		State:   "running",
		Created: 1640995200,
		Labels: map[string]string{
			"com.docker.compose.project": "myapp",
			"com.docker.compose.service": "web",
		},
		Ports: []Port{
			{
				PrivatePort: 80,
				PublicPort:  8080,
				Type:        "tcp",
				IP:          "0.0.0.0",
			},
		},
	}

	jsonData, err := json.Marshal(container)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, "abc123def456", result["Id"])
	assert.Equal(t, []interface{}{"/myapp_web_1", "/myapp_web"}, result["Names"])
	assert.Equal(t, "nginx:latest", result["Image"])
	assert.Equal(t, "Up 2 hours", result["Status"])
	assert.Equal(t, "running", result["State"])
	assert.Equal(t, float64(1640995200), result["Created"])

	// Check labels
	labels := result["Labels"].(map[string]interface{})
	assert.Equal(t, "myapp", labels["com.docker.compose.project"])
	assert.Equal(t, "web", labels["com.docker.compose.service"])

	// Check ports
	ports := result["Ports"].([]interface{})
	assert.Len(t, ports, 1)
	port := ports[0].(map[string]interface{})
	assert.Equal(t, float64(80), port["PrivatePort"])
	assert.Equal(t, float64(8080), port["PublicPort"])
	assert.Equal(t, "tcp", port["Type"])
	assert.Equal(t, "0.0.0.0", port["IP"])
}

func TestContainer_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"Id": "def456ghi789",
		"Names": ["/myapp_api_1", "/myapp_api"],
		"Image": "myapp-api:latest",
		"Status": "Up 1 hour",
		"State": "running",
		"Created": 1640998800,
		"Labels": {
			"com.docker.compose.project": "myapp",
			"com.docker.compose.service": "api"
		},
		"Ports": [
			{
				"PrivatePort": 3000,
				"PublicPort": 3001,
				"Type": "tcp",
				"IP": "127.0.0.1"
			}
		]
	}`

	var container Container
	err := json.Unmarshal([]byte(jsonData), &container)
	require.NoError(t, err)

	assert.Equal(t, "def456ghi789", container.ID)
	assert.Equal(t, []string{"/myapp_api_1", "/myapp_api"}, container.Names)
	assert.Equal(t, "myapp-api:latest", container.Image)
	assert.Equal(t, "Up 1 hour", container.Status)
	assert.Equal(t, "running", container.State)
	assert.Equal(t, int64(1640998800), container.Created)
	assert.Equal(t, "myapp", container.Labels["com.docker.compose.project"])
	assert.Equal(t, "api", container.Labels["com.docker.compose.service"])
	assert.Len(t, container.Ports, 1)
	assert.Equal(t, 3000, container.Ports[0].PrivatePort)
	assert.Equal(t, 3001, container.Ports[0].PublicPort)
	assert.Equal(t, "tcp", container.Ports[0].Type)
	assert.Equal(t, "127.0.0.1", container.Ports[0].IP)
}

func TestCreateStackRequest_JSONMarshal(t *testing.T) {
	req := CreateStackRequest{
		Name:             "myapp",
		StackFileContent: "version: '3.8'\nservices:\n  web:\n    image: nginx",
		Env: []EnvVar{
			{Name: "NODE_ENV", Value: "production"},
			{Name: "DATABASE_URL", Value: "postgres://user:pass@db:5432/myapp"},
		},
	}

	jsonData, err := json.Marshal(req)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, "myapp", result["Name"])
	assert.Equal(t, "version: '3.8'\nservices:\n  web:\n    image: nginx", result["StackFileContent"])

	// Check environment variables
	envVars := result["Env"].([]interface{})
	assert.Len(t, envVars, 2)

	env1 := envVars[0].(map[string]interface{})
	assert.Equal(t, "NODE_ENV", env1["name"])
	assert.Equal(t, "production", env1["value"])

	env2 := envVars[1].(map[string]interface{})
	assert.Equal(t, "DATABASE_URL", env2["name"])
	assert.Equal(t, "postgres://user:pass@db:5432/myapp", env2["value"])
}

func TestCreateStackRequest_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"Name": "myapp-staging",
		"StackFileContent": "version: '3.8'\nservices:\n  api:\n    image: myapp-api:staging",
		"Env": [
			{"name": "NODE_ENV", "value": "staging"},
			{"name": "API_URL", "value": "https://api.staging.example.com"}
		]
	}`

	var req CreateStackRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	require.NoError(t, err)

	assert.Equal(t, "myapp-staging", req.Name)
	assert.Equal(t, "version: '3.8'\nservices:\n  api:\n    image: myapp-api:staging", req.StackFileContent)
	assert.Len(t, req.Env, 2)
	assert.Equal(t, "NODE_ENV", req.Env[0].Name)
	assert.Equal(t, "staging", req.Env[0].Value)
	assert.Equal(t, "API_URL", req.Env[1].Name)
	assert.Equal(t, "https://api.staging.example.com", req.Env[1].Value)
}

func TestUpdateStackRequest_JSONMarshal(t *testing.T) {
	req := UpdateStackRequest{
		StackFile: "version: '3.8'\nservices:\n  web:\n    image: nginx:latest",
		PullImage: true,
		Prune:     false,
	}

	jsonData, err := json.Marshal(req)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, "version: '3.8'\nservices:\n  web:\n    image: nginx:latest", result["StackFile"])
	assert.Equal(t, true, result["PullImage"])
	assert.Equal(t, false, result["Prune"])
}

func TestUpdateStackRequest_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"StackFile": "version: '3.8'\nservices:\n  api:\n    image: myapp-api:v2.0.0",
		"PullImage": false,
		"Prune": true
	}`

	var req UpdateStackRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	require.NoError(t, err)

	assert.Equal(t, "version: '3.8'\nservices:\n  api:\n    image: myapp-api:v2.0.0", req.StackFile)
	assert.Equal(t, false, req.PullImage)
	assert.Equal(t, true, req.Prune)
}

func TestStackDetails_JSONMarshal(t *testing.T) {
	details := StackDetails{
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

	jsonData, err := json.Marshal(details)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["Id"])
	assert.Equal(t, "myapp", result["Name"])
	assert.Equal(t, float64(1), result["Status"])
	assert.Equal(t, float64(1), result["EndpointId"])
	assert.Equal(t, float64(1640995200), result["creationDate"])
	assert.Equal(t, float64(1640998800), result["updateDate"])
	assert.Equal(t, "admin", result["createdBy"])
	assert.Equal(t, "admin", result["updatedBy"])
	assert.Equal(t, "/opt/portainer/stacks/1", result["projectPath"])
	assert.Equal(t, "docker-compose.yml", result["EntryPoint"])
}

func TestStackDetails_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"Id": 2,
		"Name": "myapp-staging",
		"Status": 2,
		"EndpointId": 2,
		"creationDate": 1640995200,
		"updateDate": 1640998800,
		"createdBy": "user",
		"updatedBy": "user",
		"projectPath": "/opt/portainer/stacks/2",
		"EntryPoint": "docker-compose.staging.yml"
	}`

	var details StackDetails
	err := json.Unmarshal([]byte(jsonData), &details)
	require.NoError(t, err)

	assert.Equal(t, 2, details.ID)
	assert.Equal(t, "myapp-staging", details.Name)
	assert.Equal(t, 2, details.Status)
	assert.Equal(t, 2, details.EnvironmentID)
	assert.Equal(t, int64(1640995200), details.CreatedAt)
	assert.Equal(t, int64(1640998800), details.UpdatedAt)
	assert.Equal(t, "user", details.CreatedBy)
	assert.Equal(t, "user", details.UpdatedBy)
	assert.Equal(t, "/opt/portainer/stacks/2", details.ProjectPath)
	assert.Equal(t, "docker-compose.staging.yml", details.EntryPoint)
}

func TestAPIError_JSONMarshal(t *testing.T) {
	apiError := APIError{
		Message: "Stack not found",
		Details: "The requested stack with ID 123 does not exist",
	}

	jsonData, err := json.Marshal(apiError)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, "Stack not found", result["message"])
	assert.Equal(t, "The requested stack with ID 123 does not exist", result["details"])
}

func TestAPIError_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"message": "Invalid environment ID",
		"details": "The environment with ID 999 does not exist or is not accessible"
	}`

	var apiError APIError
	err := json.Unmarshal([]byte(jsonData), &apiError)
	require.NoError(t, err)

	assert.Equal(t, "Invalid environment ID", apiError.Message)
	assert.Equal(t, "The environment with ID 999 does not exist or is not accessible", apiError.Details)
}

func TestEnvVar_JSONMarshal(t *testing.T) {
	envVar := EnvVar{
		Name:  "DATABASE_URL",
		Value: "postgres://user:pass@localhost:5432/myapp",
	}

	jsonData, err := json.Marshal(envVar)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, "DATABASE_URL", result["name"])
	assert.Equal(t, "postgres://user:pass@localhost:5432/myapp", result["value"])
}

func TestEnvVar_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"name": "NODE_ENV",
		"value": "production"
	}`

	var envVar EnvVar
	err := json.Unmarshal([]byte(jsonData), &envVar)
	require.NoError(t, err)

	assert.Equal(t, "NODE_ENV", envVar.Name)
	assert.Equal(t, "production", envVar.Value)
}

func TestPort_JSONMarshal(t *testing.T) {
	port := Port{
		PrivatePort: 8080,
		PublicPort:  80,
		Type:        "tcp",
		IP:          "0.0.0.0",
	}

	jsonData, err := json.Marshal(port)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(8080), result["PrivatePort"])
	assert.Equal(t, float64(80), result["PublicPort"])
	assert.Equal(t, "tcp", result["Type"])
	assert.Equal(t, "0.0.0.0", result["IP"])
}

func TestPort_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"PrivatePort": 3000,
		"PublicPort": 3001,
		"Type": "tcp",
		"IP": "127.0.0.1"
	}`

	var port Port
	err := json.Unmarshal([]byte(jsonData), &port)
	require.NoError(t, err)

	assert.Equal(t, 3000, port.PrivatePort)
	assert.Equal(t, 3001, port.PublicPort)
	assert.Equal(t, "tcp", port.Type)
	assert.Equal(t, "127.0.0.1", port.IP)
}
