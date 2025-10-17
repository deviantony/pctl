# pctl - Portainer Control CLI

## Overview
pctl is a developer companion tool for deploying and managing Docker Compose applications via Portainer. It streamlines the deployment workflow by providing simple commands to create, deploy, and redeploy stacks through Portainer's API.

It aims to be a dev companion that allows a developer to quickly deploy and redeploy their application on a specific environment located in a Portainer instance.

## Core Requirements

### Commands

#### `pctl init`
- Creates a `pctl.yml` configuration file in the current directory
- Interactive form to collect:
  - Portainer URL
  - Portainer API token
  - Environment selection (auto-detected from Portainer, dropdown/list experience)
  - Stack name (default: `pctl_<project-folder-name>`, configurable)
  - Compose file path (default: `docker-compose.yml`, configurable)
- Advanced settings (TLS verification) are configured in the generated `pctl.yml` file

#### `pctl deploy`
- Creates the first deployment of the application
- Translates to stack creation in Portainer in the specified environment
- Reads configuration from `pctl.yml`
- Uses the specified Docker Compose file
- Fails if stack already exists

#### `pctl redeploy`
- Updates an existing stack deployment
- Pulls latest images and restarts services
- Requires existing stack (created via `pctl deploy`)

### Configuration File (`pctl.yml`)
```yaml
portainer_url: https://portainer.example.com
api_token: ptr_xxxxxxxxxx
environment_id: 2
stack_name: pctl_myproject
compose_file: docker-compose.yml
skip_tls_verify: true
```

### Configuration Documentation
- A comprehensive `pctl.yml.example` file is provided with detailed comments for each field
- Includes examples for different deployment scenarios (production, development, local)
- Advanced settings like TLS verification are documented but not exposed in the interactive form

### Technical Stack
- **Language**: Go
- **CLI Framework**: Cobra
- **TUI Components**: Charm ecosystem
  - Huh for interactive forms
  - Lip Gloss for styling
  - Bubble Tea for advanced TUI components (if needed)
- **Configuration**: YAML

### Defaults & Behavior
- Stack naming: `pctl_<project-folder-name>` (configurable)
- Compose file: `docker-compose.yml` (configurable)  
- API authentication: Token-based (stored plain text in config file)
- Environment detection: Auto-detect available environments, user selects from list
- TLS verification: Skip by default (recommended for self-hosted Portainer instances)

### Assumptions & Limitations
- Image building and publishing handled externally (not in scope)
- Portainer doesn't support building images, only deploying pre-built ones
- API token stored as plain text (acceptable for v1)
- Single compose file per project
- TLS certificate verification can be disabled for self-hosted instances with self-signed certificates
- **Docker Standalone environments only** - Swarm and Kubernetes environments not supported in v1

## Future Enhancements (Out of Scope for v1)
- `pctl ps` - Check stack status
- `pctl logs` - View stack logs  
- Secure token storage
- **Multi-environment type support** - Automatic detection and support for different Portainer environment types:
  - Docker Swarm environments with Swarm Compose files
  - Kubernetes environments with Kubernetes manifests
  - Environment-specific deployment logic based on detected environment type

## Success Criteria
- Developer can initialize configuration via interactive form
- Developer can deploy new stack to Portainer
- Developer can redeploy existing stack
- Clear error messages for common failure scenarios
- Configuration persists between commands