package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformComposeFile(t *testing.T) {
	originalContent := `
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

	imageTags := map[string]string{
		"web": "myapp-web:abc123",
		"api": "myapp-api:def456",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check that the result contains the expected content
	assert.Contains(t, result.TransformedContent, "version")
	assert.Contains(t, result.TransformedContent, "myapp-web:abc123")
	assert.Contains(t, result.TransformedContent, "myapp-api:def456")
	assert.NotContains(t, result.TransformedContent, "build:")

	// Check that services were modified
	assert.Len(t, result.ServicesModified, 2)
	assert.Contains(t, result.ServicesModified, "web")
	assert.Contains(t, result.ServicesModified, "api")

	// Check image tags
	assert.Equal(t, imageTags, result.ImageTags)
}

func TestTransformComposeFile_RemovesBuild(t *testing.T) {
	originalContent := `
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
`

	imageTags := map[string]string{
		"web": "myapp-web:abc123",
		"api": "myapp-api:def456",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Parse the transformed content to verify build directives were removed
	transformedCompose, err := ParseComposeFile(result.TransformedContent)
	require.NoError(t, err)

	// Check web service
	webService := transformedCompose.Services["web"].(map[string]interface{})
	assert.NotContains(t, webService, "build")
	assert.Equal(t, "myapp-web:abc123", webService["image"])

	// Check api service
	apiService := transformedCompose.Services["api"].(map[string]interface{})
	assert.NotContains(t, apiService, "build")
	assert.Equal(t, "myapp-api:def456", apiService["image"])
}

func TestTransformComposeFile_AddsImage(t *testing.T) {
	originalContent := `
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
`

	imageTags := map[string]string{
		"web": "myapp-web:abc123",
		"api": "myapp-api:def456",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Parse the transformed content to verify images were added
	transformedCompose, err := ParseComposeFile(result.TransformedContent)
	require.NoError(t, err)

	// Check web service
	webService := transformedCompose.Services["web"].(map[string]interface{})
	assert.Equal(t, "myapp-web:abc123", webService["image"])

	// Check api service
	apiService := transformedCompose.Services["api"].(map[string]interface{})
	assert.Equal(t, "myapp-api:def456", apiService["image"])
}

func TestTransformComposeFile_MultipleServices(t *testing.T) {
	originalContent := `
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
  worker:
    build:
      context: ./worker
      dockerfile: Dockerfile.worker
    environment:
      - WORKER_ENV=production
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`

	imageTags := map[string]string{
		"web":    "myapp-web:abc123",
		"api":    "myapp-api:def456",
		"worker": "myapp-worker:ghi789",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check that all services were modified
	assert.Len(t, result.ServicesModified, 3)
	assert.Contains(t, result.ServicesModified, "web")
	assert.Contains(t, result.ServicesModified, "api")
	assert.Contains(t, result.ServicesModified, "worker")

	// Parse the transformed content to verify all transformations
	transformedCompose, err := ParseComposeFile(result.TransformedContent)
	require.NoError(t, err)

	// Check web service
	webService := transformedCompose.Services["web"].(map[string]interface{})
	assert.Equal(t, "myapp-web:abc123", webService["image"])
	assert.NotContains(t, webService, "build")

	// Check api service
	apiService := transformedCompose.Services["api"].(map[string]interface{})
	assert.Equal(t, "myapp-api:def456", apiService["image"])
	assert.NotContains(t, apiService, "build")

	// Check worker service
	workerService := transformedCompose.Services["worker"].(map[string]interface{})
	assert.Equal(t, "myapp-worker:ghi789", workerService["image"])
	assert.NotContains(t, workerService, "build")

	// Check that db service was not modified
	dbService := transformedCompose.Services["db"].(map[string]interface{})
	assert.Equal(t, "postgres:13", dbService["image"])
	assert.NotContains(t, dbService, "build")
}

func TestTransformComposeFile_ServiceNotFound(t *testing.T) {
	originalContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
`

	imageTags := map[string]string{
		"nonexistent": "myapp-nonexistent:abc123",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "service 'nonexistent' not found in compose file")
}

func TestTransformResult_ValidateTransformation(t *testing.T) {
	originalContent := `
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
`

	imageTags := map[string]string{
		"web": "myapp-web:abc123",
		"api": "myapp-api:def456",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)

	// Validate the transformation
	err = result.ValidateTransformation()
	assert.NoError(t, err)
}

func TestTransformResult_ValidateTransformation_BuildRemaining(t *testing.T) {
	// Create a result with build directives still present (simulated)
	result := &TransformResult{
		TransformedContent: `
version: '3.8'
services:
  web:
    build: .
    image: myapp-web:abc123
    ports:
      - "3000:3000"
`,
		ImageTags: map[string]string{
			"web": "myapp-web:abc123",
		},
		ServicesModified: []string{"web"},
	}

	// This should fail validation because build directive is still present
	err := result.ValidateTransformation()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "still has build directive after transformation")
}

func TestTransformResult_ValidateTransformation_ImageMissing(t *testing.T) {
	// Create a result with missing image (simulated)
	result := &TransformResult{
		TransformedContent: `
version: '3.8'
services:
  web:
    ports:
      - "3000:3000"
`,
		ImageTags: map[string]string{
			"web": "myapp-web:abc123",
		},
		ServicesModified: []string{"web"},
	}

	// This should fail validation because image is missing
	err := result.ValidateTransformation()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing image after transformation")
}

func TestTransformResult_ValidateTransformation_WrongImage(t *testing.T) {
	// Create a result with wrong image tag (simulated)
	result := &TransformResult{
		TransformedContent: `
version: '3.8'
services:
  web:
    image: wrong-tag:wrong
    ports:
      - "3000:3000"
`,
		ImageTags: map[string]string{
			"web": "myapp-web:abc123",
		},
		ServicesModified: []string{"web"},
	}

	// This should fail validation because image tag is wrong
	err := result.ValidateTransformation()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incorrect image tag")
}

func TestTransformResult_GetTransformationSummary(t *testing.T) {
	originalContent := `
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
`

	imageTags := map[string]string{
		"web": "myapp-web:abc123",
		"api": "myapp-api:def456",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)

	summary := result.GetTransformationSummary()
	assert.Contains(t, summary, "Transformed 2 service(s)")
	assert.Contains(t, summary, "web: build -> image: myapp-web:abc123")
	assert.Contains(t, summary, "api: build -> image: myapp-api:def456")
}

func TestDiffTransformation(t *testing.T) {
	originalContent := `
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

	transformedContent := `
version: '3.8'
services:
  web:
    image: myapp-web:abc123
    ports:
      - "3000:3000"
  api:
    image: myapp-api:def456
    ports:
      - "8080:8080"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`

	diff, err := DiffTransformation(originalContent, transformedContent)
	require.NoError(t, err)

	assert.Contains(t, diff, "Compose file transformation diff")
	assert.Contains(t, diff, "web: build directive removed")
	assert.Contains(t, diff, "api: build directive removed")
	assert.Contains(t, diff, "web: image added: myapp-web:abc123")
	assert.Contains(t, diff, "api: image added: myapp-api:def456")
}

func TestTransformComposeFile_EmptyImageTags(t *testing.T) {
	originalContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  db:
    image: postgres:13
`

	imageTags := map[string]string{}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return content with no modifications (YAML formatting may differ)
	assert.Empty(t, result.ServicesModified)
	assert.Empty(t, result.ImageTags)

	// Parse both to compare structure
	originalCompose, err := ParseComposeFile(originalContent)
	require.NoError(t, err)
	transformedCompose, err := ParseComposeFile(result.TransformedContent)
	require.NoError(t, err)

	// Should have same services and version
	assert.Equal(t, originalCompose.Version, transformedCompose.Version)
	assert.Equal(t, len(originalCompose.Services), len(transformedCompose.Services))
}

func TestTransformComposeFile_InvalidServiceDefinition(t *testing.T) {
	originalContent := `
version: '3.8'
services:
  web: "invalid-service-definition"
`

	imageTags := map[string]string{
		"web": "myapp-web:abc123",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "is not a valid service definition")
}

func TestTransformResult_GetTransformationSummary_NoServices(t *testing.T) {
	result := &TransformResult{
		TransformedContent: "version: '3.8'\nservices: {}",
		ImageTags:          map[string]string{},
		ServicesModified:   []string{},
	}

	summary := result.GetTransformationSummary()
	assert.Equal(t, "No services were transformed", summary)
}

func TestTransformComposeFile_PreservesOtherFields(t *testing.T) {
	originalContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
    volumes:
      - ./src:/app/src
    networks:
      - app-network
    depends_on:
      - db
`

	imageTags := map[string]string{
		"web": "myapp-web:abc123",
	}

	result, err := TransformComposeFile(originalContent, imageTags)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Parse the transformed content to verify other fields are preserved
	transformedCompose, err := ParseComposeFile(result.TransformedContent)
	require.NoError(t, err)

	webService := transformedCompose.Services["web"].(map[string]interface{})

	// Check that image was added
	assert.Equal(t, "myapp-web:abc123", webService["image"])

	// Check that other fields are preserved
	assert.Contains(t, webService, "ports")
	assert.Contains(t, webService, "environment")
	assert.Contains(t, webService, "volumes")
	assert.Contains(t, webService, "networks")
	assert.Contains(t, webService, "depends_on")

	// Check that build directive was removed
	assert.NotContains(t, webService, "build")
}
