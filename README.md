# pctl - Portainer Control CLI

A developer companion tool for deploying and managing Docker Compose applications via Portainer. pctl streamlines the deployment workflow by providing simple commands to create, deploy, and redeploy stacks through Portainer's API.

## Features

- 🚀 **Easy Deployment**: Deploy Docker Compose stacks to Portainer with a single command
- 🔄 **Quick Redeployment**: Update existing stacks with latest images
- 📊 **Stack Monitoring**: View stack status and running containers
- 📝 **Interactive Setup**: Guided configuration with beautiful terminal UI
- 🔧 **Flexible Configuration**: Support for multiple environments and custom compose files
- 📋 **Logs Viewer**: Real-time log streaming from your containers

## Installation

### From Source

1. Clone the repository:
```bash
git clone <repository-url>
cd pctl
```

2. Build the binary:
```bash
make build
```

3. Install the binary (optional):
```bash
sudo cp build/pctl /usr/local/bin/
```

### Cross-Platform Builds

Build for multiple platforms:
```bash
make build-all
```

This creates binaries for:
- Linux (AMD64/ARM64)
- macOS (AMD64/ARM64)
- Windows (AMD64)

## Quick Start

### 1. Initialize Configuration

Run the interactive setup to configure pctl for your Portainer instance:

```bash
pctl init
```

This will guide you through:
- Portainer URL (e.g., `https://portainer.example.com`)
- API Token (generate in Portainer: Settings > API Keys)
- Environment selection (auto-detected from your Portainer)
- Stack name (defaults to `pctl_<project-folder-name>`)
- Compose file path (defaults to `docker-compose.yml`)

### 2. Deploy Your Application

Deploy a new stack to Portainer:

```bash
pctl deploy
```

This command will:
- Read your `docker-compose.yml` file
- Create a new stack in Portainer
- Deploy all services defined in your compose file

### 3. Check Status

View your stack status and running containers:

```bash
pctl ps
```

### 4. Redeploy Updates

Update an existing stack with latest images:

```bash
pctl redeploy
```

### 5. View Logs

Stream logs from your stack containers:

```bash
pctl logs
```

This command provides:
- Real-time log streaming
- Interactive log viewer with search functionality
- Support for multiple containers

## Configuration

pctl uses a `pctl.yml` configuration file in your project directory. Here's an example:

```yaml
# Portainer instance URL
portainer_url: https://portainer.example.com

# Portainer API token (generate in Portainer: Settings > API Keys)
api_token: ptr_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Portainer environment ID
environment_id: 1

# Stack name in Portainer
stack_name: pctl_myproject

# Docker Compose file path
compose_file: docker-compose.yml

# Skip TLS certificate verification (recommended for self-hosted)
skip_tls_verify: true
```

### Configuration Examples

**Production Setup:**
```yaml
portainer_url: https://portainer.company.com
api_token: ptr_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
environment_id: 1
stack_name: pctl_production_app
compose_file: docker-compose.prod.yml
skip_tls_verify: false
```

**Development Setup:**
```yaml
portainer_url: https://192.168.1.100:9443
api_token: ptr_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
environment_id: 2
stack_name: pctl_dev_app
compose_file: docker-compose.dev.yml
skip_tls_verify: true
```

## Commands

### `pctl init`
Initialize pctl configuration with an interactive form.

**Features:**
- Validates Portainer URL format
- Validates API token format (must start with `ptr_`)
- Auto-detects available environments from Portainer
- Provides sensible defaults for stack name and compose file

### `pctl deploy`
Deploy a new Docker Compose stack to Portainer.

**What it does:**
- Reads your `docker-compose.yml` file
- Creates a new stack in the configured Portainer environment
- Fails if stack already exists (use `pctl redeploy` instead)

### `pctl redeploy`
Update an existing stack deployment.

**What it does:**
- Pulls latest images for all services
- Restarts services with updated images
- Requires an existing stack (created via `pctl deploy`)

### `pctl ps`
Show stack status and running containers.

**Displays:**
- Stack information and status
- Container details (name, image, status, ports)
- Resource usage information
- Real-time container status

### `pctl logs`
View and stream logs from stack containers.

**Features:**
- Real-time log streaming
- Interactive log viewer with search functionality
- Support for multiple containers
- Color-coded log levels
- Navigate between different containers
- Search and filter log content

## Requirements

- **Go 1.25.1+** (for building from source)
- **Portainer instance** with API access
- **Docker Compose file** in your project
- **Portainer API token** (generate in Portainer: Settings > API Keys)

## Supported Environments

- **Docker Standalone** environments (primary support)
- **Docker Swarm** environments (limited support)
- **Kubernetes** environments (limited support)

> **Note**: Full support for Swarm and Kubernetes environments is planned for future versions.

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install dependencies
make deps
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Linting

```bash
make lint
```

## Project Structure

```
pctl/
├── cmd/                    # CLI commands
│   ├── deploy/            # Deploy command
│   ├── init/              # Init command
│   ├── logs/              # Logs command
│   ├── ps/                # Status command
│   └── redeploy/          # Redeploy command
├── internal/              # Internal packages
│   ├── compose/           # Docker Compose handling
│   ├── config/            # Configuration management
│   └── portainer/         # Portainer API client
├── data/                  # Example data files
├── build/                 # Build output
├── pctl.yml.example       # Configuration example
└── Makefile              # Build automation
```

## Troubleshooting

### Common Issues

**"Configuration file not found"**
- Run `pctl init` to create the configuration file

**"Stack already exists"**
- Use `pctl redeploy` instead of `pctl deploy` for existing stacks

**"Invalid API token"**
- Generate a new API token in Portainer: Settings > API Keys
- Ensure the token starts with `ptr_`

**"TLS certificate verification failed"**
- Set `skip_tls_verify: true` in your `pctl.yml` for self-hosted Portainer instances

**"No environments found"**
- Ensure your Portainer user has access to environments
- Check that your API token has sufficient permissions

### Getting Help

1. Check your `pctl.yml` configuration
2. Verify your Portainer API token permissions
3. Ensure your Docker Compose file is valid
4. Check Portainer logs for detailed error messages

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Charm](https://charm.sh/) - Terminal UI components
- [Go](https://golang.org/) - Programming language
