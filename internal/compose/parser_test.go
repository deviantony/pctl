package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComposeFile(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`

	compose, err := ParseComposeFile(composeContent)
	require.NoError(t, err)
	require.NotNil(t, compose)

	assert.Equal(t, "3.8", compose.Version)
	assert.NotNil(t, compose.Services)
	assert.Contains(t, compose.Services, "web")
	assert.Contains(t, compose.Services, "db")
}

func TestParseComposeFile_InvalidYAML(t *testing.T) {
	invalidYAML := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  invalid_yaml: [unclosed array
`

	compose, err := ParseComposeFile(invalidYAML)
	assert.Error(t, err)
	assert.Nil(t, compose)
	assert.Contains(t, err.Error(), "failed to parse compose file")
}

func TestComposeFile_FindServicesWithBuild(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  api:
    build:
      context: ./api
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`

	compose, err := ParseComposeFile(composeContent)
	require.NoError(t, err)

	servicesWithBuild, err := compose.FindServicesWithBuild()
	require.NoError(t, err)

	assert.Len(t, servicesWithBuild, 2)

	// Check web service
	webService := findServiceByName(servicesWithBuild, "web")
	require.NotNil(t, webService)
	assert.Equal(t, "web", webService.ServiceName)
	assert.NotNil(t, webService.Build)
	assert.Equal(t, ".", webService.Build.Context)
	assert.Equal(t, "Dockerfile", webService.Build.Dockerfile)

	// Check api service
	apiService := findServiceByName(servicesWithBuild, "api")
	require.NotNil(t, apiService)
	assert.Equal(t, "api", apiService.ServiceName)
	assert.NotNil(t, apiService.Build)
	assert.Equal(t, "./api", apiService.Build.Context)
	assert.Equal(t, "Dockerfile.api", apiService.Build.Dockerfile)
}

func TestComposeFile_FindServicesWithBuild_None(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`

	compose, err := ParseComposeFile(composeContent)
	require.NoError(t, err)

	servicesWithBuild, err := compose.FindServicesWithBuild()
	require.NoError(t, err)

	assert.Empty(t, servicesWithBuild)
}

func TestExtractBuildInfo_StringFormat(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": "./src",
		"ports": []interface{}{"3000:3000"},
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "web", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, "./src", buildInfo.Build.Context)
	assert.Equal(t, "Dockerfile", buildInfo.Build.Dockerfile) // Default
}

func TestExtractBuildInfo_MapFormat(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": map[string]interface{}{
			"context":    "./api",
			"dockerfile": "Dockerfile.api",
			"args": map[string]interface{}{
				"NODE_ENV": "production",
				"VERSION":  "1.0.0",
			},
			"target": "production",
			"cache_from": []interface{}{
				"myapp:latest",
				"myapp:dev",
			},
		},
		"ports": []interface{}{"8080:8080"},
	}

	buildInfo, err := extractBuildInfo("api", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "api", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, "./api", buildInfo.Build.Context)
	assert.Equal(t, "Dockerfile.api", buildInfo.Build.Dockerfile)
	assert.Equal(t, "production", buildInfo.Build.Target)

	// Check args
	assert.Len(t, buildInfo.Build.Args, 2)
	assert.Equal(t, "production", buildInfo.Build.Args["NODE_ENV"])
	assert.Equal(t, "1.0.0", buildInfo.Build.Args["VERSION"])

	// Check cache_from
	assert.Len(t, buildInfo.Build.CacheFrom, 2)
	assert.Contains(t, buildInfo.Build.CacheFrom, "myapp:latest")
	assert.Contains(t, buildInfo.Build.CacheFrom, "myapp:dev")
}

func TestExtractBuildInfo_WithArgs(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": map[string]interface{}{
			"context": "./src",
			"args": map[string]interface{}{
				"BUILD_ENV": "production",
				"VERSION":   "2.0.0",
			},
		},
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "web", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, "./src", buildInfo.Build.Context)
	assert.Len(t, buildInfo.Build.Args, 2)
	assert.Equal(t, "production", buildInfo.Build.Args["BUILD_ENV"])
	assert.Equal(t, "2.0.0", buildInfo.Build.Args["VERSION"])
}

func TestExtractBuildInfo_WithTarget(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": map[string]interface{}{
			"context": "./src",
			"target":  "production",
		},
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "web", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, "./src", buildInfo.Build.Context)
	assert.Equal(t, "production", buildInfo.Build.Target)
}

func TestExtractBuildInfo_WithCacheFrom(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": map[string]interface{}{
			"context": "./src",
			"cache_from": []interface{}{
				"myapp:latest",
				"myapp:dev",
				"myapp:test",
			},
		},
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "web", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, "./src", buildInfo.Build.Context)
	assert.Len(t, buildInfo.Build.CacheFrom, 3)
	assert.Contains(t, buildInfo.Build.CacheFrom, "myapp:latest")
	assert.Contains(t, buildInfo.Build.CacheFrom, "myapp:dev")
	assert.Contains(t, buildInfo.Build.CacheFrom, "myapp:test")
}

func TestExtractBuildInfo_DefaultDockerfile(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": "./src",
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "web", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, "./src", buildInfo.Build.Context)
	assert.Equal(t, "Dockerfile", buildInfo.Build.Dockerfile) // Default
}

func TestComposeFile_HasBuildDirectives(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  db:
    image: postgres:13
`

	compose, err := ParseComposeFile(composeContent)
	require.NoError(t, err)

	hasBuild, err := compose.HasBuildDirectives()
	require.NoError(t, err)
	assert.True(t, hasBuild)
}

func TestComposeFile_GetServiceNames(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  api:
    build: ./api
    ports:
      - "8080:8080"
  db:
    image: postgres:13
`

	compose, err := ParseComposeFile(composeContent)
	require.NoError(t, err)

	serviceNames := compose.GetServiceNames()
	assert.Len(t, serviceNames, 3)
	assert.Contains(t, serviceNames, "web")
	assert.Contains(t, serviceNames, "api")
	assert.Contains(t, serviceNames, "db")
}

func TestComposeFile_GetBuildContextSummary(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  api:
    build:
      context: ./api
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
  db:
    image: postgres:13
`

	compose, err := ParseComposeFile(composeContent)
	require.NoError(t, err)

	summary, err := compose.GetBuildContextSummary()
	require.NoError(t, err)

	assert.Contains(t, summary, "Found 2 service(s) with build directives")
	assert.Contains(t, summary, "web: context=., dockerfile=Dockerfile")
	assert.Contains(t, summary, "api: context=./api, dockerfile=Dockerfile.api")
}

func TestComposeFile_GetBuildContextSummary_NoBuilds(t *testing.T) {
	composeContent := `
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  db:
    image: postgres:13
`

	compose, err := ParseComposeFile(composeContent)
	require.NoError(t, err)

	summary, err := compose.GetBuildContextSummary()
	require.NoError(t, err)

	assert.Equal(t, "No build directives found", summary)
}

func TestExtractBuildInfo_InvalidFormat(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": 123, // Invalid type
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	assert.Error(t, err)
	assert.Nil(t, buildInfo)
	assert.Contains(t, err.Error(), "invalid build directive format")
}

func TestExtractBuildInfo_NoBuild(t *testing.T) {
	serviceData := map[string]interface{}{
		"image": "nginx:latest",
		"ports": []interface{}{"80:80"},
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	assert.Nil(t, buildInfo)
}

func TestExtractBuildInfo_NonMapService(t *testing.T) {
	serviceData := "nginx:latest" // Not a map

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	assert.Nil(t, buildInfo)
}

func TestExtractBuildInfo_EmptyContext(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": map[string]interface{}{
			"context": "", // Empty context
		},
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "web", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, ".", buildInfo.Build.Context) // Should default to "."
}

func TestExtractBuildInfo_WithComplexArgs(t *testing.T) {
	serviceData := map[string]interface{}{
		"build": map[string]interface{}{
			"context": "./src",
			"args": map[string]interface{}{
				"BUILD_ENV":    "production",
				"VERSION":      "1.0.0",
				"NODE_VERSION": "18",
				"REGISTRY":     "myregistry.com",
			},
		},
	}

	buildInfo, err := extractBuildInfo("web", serviceData)
	require.NoError(t, err)
	require.NotNil(t, buildInfo)

	assert.Equal(t, "web", buildInfo.ServiceName)
	assert.NotNil(t, buildInfo.Build)
	assert.Equal(t, "./src", buildInfo.Build.Context)
	assert.Len(t, buildInfo.Build.Args, 4)
	assert.Equal(t, "production", buildInfo.Build.Args["BUILD_ENV"])
	assert.Equal(t, "1.0.0", buildInfo.Build.Args["VERSION"])
	assert.Equal(t, "18", buildInfo.Build.Args["NODE_VERSION"])
	assert.Equal(t, "myregistry.com", buildInfo.Build.Args["REGISTRY"])
}

// Helper function to find a service by name in the slice
func findServiceByName(services []ServiceBuildInfo, name string) *ServiceBuildInfo {
	for _, service := range services {
		if service.ServiceName == name {
			return &service
		}
	}
	return nil
}
