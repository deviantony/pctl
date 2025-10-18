# pctl - Portainer Control CLI

A simple CLI tool for deploying and managing Docker Compose applications via Portainer. Deploy your stacks with a single command and manage them through Portainer's API.

[![asciicast](https://asciinema.org/a/zYM6Tu31LesuRowrLDJZfGzcU.svg)](https://asciinema.org/a/zYM6Tu31LesuRowrLDJZfGzcU)

## Usage

### Prerequisites
- A `docker-compose.yml` file in your project directory
- Portainer instance with API access
- Portainer API token (generate in Portainer: Settings > API Keys)

### 1. Initialize Configuration
```bash
pctl init
```
Interactive setup to configure your Portainer connection (URL, API token, environment). This creates a `pctl.yml` configuration file.

### 2. Deploy Your Application
```bash
pctl deploy
```
Deploy your Docker Compose stack to Portainer. The tool reads your `docker-compose.yml` file and creates a new stack.

### 3. Update Existing Stack
```bash
pctl redeploy
```
Update an existing stack with latest images.

### 4. Check Status
```bash
pctl ps
```
View stack status and running containers.

### 5. View Logs
```bash
pctl logs
```
Stream real-time logs from your containers.

## Configuration

pctl uses a `pctl.yml` configuration file created during initialization:

```yaml
portainer_url: https://portainer.example.com
api_token: ptr_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
environment_id: 1
stack_name: pctl_myproject
compose_file: docker-compose.yml
skip_tls_verify: true
```

The configuration includes:
- **portainer_url**: Your Portainer instance URL
- **api_token**: Portainer API token (starts with `ptr_`)
- **environment_id**: Portainer environment ID
- **stack_name**: Name for your stack in Portainer
- **compose_file**: Path to your Docker Compose file
- **skip_tls_verify**: Skip TLS verification for self-hosted instances

## Installation

Download the latest release for your platform from [GitHub Releases](https://github.com/deviantony/pctl/releases/tag/v1.0.0):

### Linux
```bash
# AMD64
wget https://github.com/deviantony/pctl/releases/download/v1.0.0/pctl_1.0.0_linux_amd64
chmod +x pctl_1.0.0_linux_amd64
sudo mv pctl_1.0.0_linux_amd64 /usr/local/bin/pctl

# ARM64
wget https://github.com/deviantony/pctl/releases/download/v1.0.0/pctl_1.0.0_linux_arm64
chmod +x pctl_1.0.0_linux_arm64
sudo mv pctl_1.0.0_linux_arm64 /usr/local/bin/pctl
```

### macOS
```bash
# AMD64
wget https://github.com/deviantony/pctl/releases/download/v1.0.0/pctl_1.0.0_darwin_amd64
chmod +x pctl_1.0.0_darwin_amd64
sudo mv pctl_1.0.0_darwin_amd64 /usr/local/bin/pctl

# ARM64 (Apple Silicon)
wget https://github.com/deviantony/pctl/releases/download/v1.0.0/pctl_1.0.0_darwin_arm64
chmod +x pctl_1.0.0_darwin_arm64
sudo mv pctl_1.0.0_darwin_arm64 /usr/local/bin/pctl
```

### Windows
```bash
# AMD64
wget https://github.com/deviantony/pctl/releases/download/v1.0.0/pctl_1.0.0_windows_amd64.exe
# Move pctl_1.0.0_windows_amd64.exe to your PATH and rename to pctl.exe
```

## Limitations

- **Docker Standalone environments only** - Full support for Kubernetes environments is planned for future versions.
