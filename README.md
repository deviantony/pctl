# pctl - Dev Companion for Portainer

A simple CLI tool for quickly deploying and re-deploying your application on a Portainer environment. 

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

**Build Support**: If your compose file contains `build:` directives, pctl will automatically build the images before deployment. See the [Build Configuration](#build-configuration) section for details.

### 3. Update Existing Stack
```bash
pctl redeploy
```
Update an existing stack with latest images.

**Force Rebuild**: Use the `-f` or `--force-rebuild` flag to force rebuild images even if they haven't changed:
```bash
pctl redeploy -f
```
This sets `force_build=true` for this run, which includes no-cache behavior, ensuring a complete rebuild of all images.

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

**Log Options**:
- `-t, --tail N`: Show the last N lines from the end of logs (default: 50)
- `-s, --service NAME`: Show logs from a specific service only
- `--non-interactive`: Force non-interactive mode (useful for testing and automation)

Examples:
```bash
# Show last 100 lines from all containers
pctl logs -t 100

# Show logs from a specific service
pctl logs -s web

# Show last 20 lines from the database service
pctl logs -s database -t 20

# Force non-interactive mode (useful for testing)
pctl logs --non-interactive

# Non-interactive mode with specific service and tail count
pctl logs -s web -t 10 --non-interactive
```

### 6. Check Version
```bash
pctl version
```
Display version information including version number, git commit hash, build timestamp, Go version, and target platform.

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

## Build Configuration

When using `build:` directives in your compose file, pctl can automatically build images before deployment. Add a `build` section to your `pctl.yml`:

```yaml
build:
  mode: remote-build        # remote-build (default) or load
  parallel: auto            # concurrent builds (auto or number)
  tag_format: "pctl-{{stack}}-{{service}}:{{hash}}"
  platforms: ["linux/amd64"]  # for load mode
  extra_build_args: {}      # global build args
  force_build: false        # force rebuild even if unchanged
  warn_threshold_mb: 50     # warn if context > 50MB
```

### Build Modes

- **remote-build** (default): Builds images on the remote Docker engine via Portainer's Docker proxy. Most bandwidth-efficient.
- **load**: Builds images locally and uploads them to the remote engine. Useful when the remote has poor internet access.

### Example Compose with Build

```yaml
version: '3.8'
services:
  web:
    build:
      context: ./web
      dockerfile: Dockerfile
      args:
        NODE_ENV: production
    ports:
      - "3000:3000"
  
  api:
    build: ./api
    ports:
      - "8080:8080"
```

When you run `pctl deploy`, it will:
1. Detect the `build:` directives
2. Build the images according to your build configuration
3. Transform the compose file to use the built images
4. Deploy the stack to Portainer

## Installation

Download the latest release for your platform from [GitHub Releases](https://github.com/deviantony/pctl/releases/latest):

### Linux AMD64
```bash
wget https://github.com/deviantony/pctl/releases/latest/download/pctl_1.2.0_linux_amd64.tar.gz
tar -xzf pctl_1.2.0_linux_amd64.tar.gz
chmod +x pctl
sudo mv pctl /usr/local/bin/
```

### Linux ARM64
```bash
wget https://github.com/deviantony/pctl/releases/latest/download/pctl_1.2.0_linux_arm64.tar.gz
tar -xzf pctl_1.2.0_linux_arm64.tar.gz
chmod +x pctl
sudo mv pctl /usr/local/bin/
```

### macOS AMD64
```bash
wget https://github.com/deviantony/pctl/releases/latest/download/pctl_1.2.0_darwin_amd64.tar.gz
tar -xzf pctl_1.2.0_darwin_amd64.tar.gz
chmod +x pctl
sudo mv pctl /usr/local/bin/
```

### macOS ARM64 (Apple Silicon)
```bash
wget https://github.com/deviantony/pctl/releases/latest/download/pctl_1.2.0_darwin_arm64.tar.gz
tar -xzf pctl_1.2.0_darwin_arm64.tar.gz
chmod +x pctl
sudo mv pctl /usr/local/bin/
```

### Windows AMD64
```bash
wget https://github.com/deviantony/pctl/releases/latest/download/pctl_1.2.0_windows_amd64.zip
# Extract the zip file and move pctl.exe to your PATH
```

## Testing

pctl includes both unit tests and integration tests to ensure reliability and correctness.

### Unit Tests

Run the unit test suite:

```bash
# Run all unit tests
make test

# Run tests with coverage report
make test-coverage
```

Unit tests cover:
- Configuration loading and validation
- Compose file parsing and transformation
- Build system components (tagging, context handling, orchestration)
- Portainer API client functionality
- Error handling and formatting

### Integration Tests

Integration tests run against a real Portainer instance to validate end-to-end functionality.

**Prerequisites:**
- Access to a Portainer instance
- Portainer API token
- Valid environment ID in Portainer

**Setup:**

1. Copy the example configuration:
```bash
cp integration_test_config.json.example integration_test_config.json
```

2. Edit `integration_test_config.json` with your Portainer details:
```json
{
  "portainer_url": "https://your-portainer-instance.com",
  "api_token": "ptr_your_api_token_here",
  "environment_id": 1
}
```

**Running Integration Tests:**

```bash
# Run integration tests
make test-integration
```

Integration tests cover:
- Deploying simple stacks (images only)
- Redeploying existing stacks
- Force rebuild with `-f` flag
- Stack status checking (`pctl ps`)
- Container logs (`pctl logs`)
- Build functionality (both remote-build and load modes)
- Error handling for non-existent stacks

**Note:** Integration tests will create and clean up test stacks automatically. The tests use unique stack names to avoid conflicts.

## Development

### Creating Releases

Releases are automated using [GoReleaser](https://goreleaser.com/) and GitHub Actions. To create a new release:

```bash
# Create a new release (e.g., version 1.1.1)
./scripts/release.sh 1.1.1

# Dry run to see what would happen
./scripts/release.sh 1.1.1 --dry-run
```

This will:
1. Create a git tag `v1.1.1`
2. Push the tag to GitHub
3. GoReleaser automatically builds binaries for all platforms
4. Creates a GitHub release with all binaries, checksums, and release notes

See [RELEASE.md](RELEASE.md) for detailed release process documentation.

## Limitations

- **Docker Standalone environments only** - Full support for Kubernetes environments is planned for future versions.