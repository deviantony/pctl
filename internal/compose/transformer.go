package compose

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// TransformResult represents the result of transforming a compose file
type TransformResult struct {
	TransformedContent string
	ImageTags          map[string]string // service name -> image tag
	ServicesModified   []string          // list of services that were modified
}

// TransformComposeFile transforms a compose file by replacing build directives with image references
func TransformComposeFile(originalContent string, imageTags map[string]string) (*TransformResult, error) {
	// Parse the original compose file
	compose, err := ParseComposeFile(originalContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Create a copy of the services map for modification
	transformedServices := make(map[string]interface{})
	for name, service := range compose.Services {
		transformedServices[name] = service
	}

	var servicesModified []string

	// Transform each service that has a corresponding image tag
	for serviceName, imageTag := range imageTags {
		serviceData, exists := transformedServices[serviceName]
		if !exists {
			return nil, fmt.Errorf("service '%s' not found in compose file", serviceName)
		}

		serviceMap, ok := serviceData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("service '%s' is not a valid service definition", serviceName)
		}

		// Remove build directive and add image
		delete(serviceMap, "build")
		serviceMap["image"] = imageTag

		servicesModified = append(servicesModified, serviceName)
	}

	// Create the transformed compose structure
	transformedCompose := ComposeFile{
		Services: transformedServices,
		Version:  compose.Version,
		Volumes:  compose.Volumes,
		Networks: compose.Networks,
	}

	// Marshal back to YAML
	transformedBytes, err := yaml.Marshal(transformedCompose)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed compose file: %w", err)
	}

	return &TransformResult{
		TransformedContent: string(transformedBytes),
		ImageTags:          imageTags,
		ServicesModified:   servicesModified,
	}, nil
}

// ValidateTransformation validates that the transformation was successful
func (tr *TransformResult) ValidateTransformation() error {
	// Parse the transformed content to ensure it's valid
	transformedCompose, err := ParseComposeFile(tr.TransformedContent)
	if err != nil {
		return fmt.Errorf("transformed compose file is invalid: %w", err)
	}

	// Check that all services with image tags have been transformed
	for serviceName, imageTag := range tr.ImageTags {
		serviceData, exists := transformedCompose.Services[serviceName]
		if !exists {
			return fmt.Errorf("service '%s' missing from transformed compose file", serviceName)
		}

		serviceMap, ok := serviceData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("service '%s' is not a valid service definition in transformed compose", serviceName)
		}

		// Check that build directive was removed
		if _, hasBuild := serviceMap["build"]; hasBuild {
			return fmt.Errorf("service '%s' still has build directive after transformation", serviceName)
		}

		// Check that image was set correctly
		image, hasImage := serviceMap["image"]
		if !hasImage {
			return fmt.Errorf("service '%s' missing image after transformation", serviceName)
		}

		imageStr, ok := image.(string)
		if !ok {
			return fmt.Errorf("service '%s' has invalid image type after transformation", serviceName)
		}

		if imageStr != imageTag {
			return fmt.Errorf("service '%s' has incorrect image tag: expected '%s', got '%s'",
				serviceName, imageTag, imageStr)
		}
	}

	return nil
}

// GetTransformationSummary returns a summary of the transformation for logging
func (tr *TransformResult) GetTransformationSummary() string {
	if len(tr.ServicesModified) == 0 {
		return "No services were transformed"
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Transformed %d service(s):\n", len(tr.ServicesModified)))

	for _, serviceName := range tr.ServicesModified {
		imageTag := tr.ImageTags[serviceName]
		summary.WriteString(fmt.Sprintf("  - %s: build -> image: %s\n", serviceName, imageTag))
	}

	return summary.String()
}

// DiffTransformation shows the differences between original and transformed compose files
func DiffTransformation(originalContent, transformedContent string) (string, error) {
	// Parse both files
	original, err := ParseComposeFile(originalContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse original compose file: %w", err)
	}

	transformed, err := ParseComposeFile(transformedContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse transformed compose file: %w", err)
	}

	var diff strings.Builder
	diff.WriteString("Compose file transformation diff:\n")

	// Check each service
	for serviceName, originalService := range original.Services {
		transformedService, exists := transformed.Services[serviceName]
		if !exists {
			diff.WriteString(fmt.Sprintf("  - %s: REMOVED\n", serviceName))
			continue
		}

		originalMap, ok1 := originalService.(map[string]interface{})
		transformedMap, ok2 := transformedService.(map[string]interface{})

		if !ok1 || !ok2 {
			continue
		}

		// Check for build directive removal
		if _, hasBuild := originalMap["build"]; hasBuild {
			if _, hasBuildAfter := transformedMap["build"]; !hasBuildAfter {
				diff.WriteString(fmt.Sprintf("  - %s: build directive removed\n", serviceName))
			}
		}

		// Check for image addition
		if _, hasImage := originalMap["image"]; !hasImage {
			if image, hasImageAfter := transformedMap["image"]; hasImageAfter {
				diff.WriteString(fmt.Sprintf("  - %s: image added: %v\n", serviceName, image))
			}
		}
	}

	return diff.String(), nil
}

