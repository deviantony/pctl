# Release Process

This document explains how to create new releases for pctl using the automated GitHub Actions workflow.

## Quick Start

To create a new release, simply run:

```bash
./scripts/release.sh 1.2.0
```

This will:
1. Create a git tag `v1.2.0`
2. Push the tag to GitHub
3. Trigger the automated release workflow
4. Build binaries for all platforms
5. Create a GitHub release with all binaries and release notes

## Prerequisites

- You must be on the `main` or `master` branch
- Your working directory must be clean (no uncommitted changes)
- You must have push access to the repository

## Release Workflow

The automated release process includes:

### 1. Build Process
- Builds binaries for multiple platforms:
  - Linux AMD64 (`pctl-linux-amd64`)
  - Linux ARM64 (`pctl-linux-arm64`)
  - macOS AMD64 (`pctl-darwin-amd64`)
  - macOS ARM64 (`pctl-darwin-arm64`)
  - Windows AMD64 (`pctl-windows-amd64.exe`)

### 2. Release Creation
- Creates a GitHub release with the tag name
- Generates automatic release notes from git commits
- Uploads all built binaries
- Creates and uploads checksums file

### 3. Release Notes
The workflow automatically generates release notes including:
- List of commits since the last release
- Installation instructions for all platforms
- Link to the full changelog

## Manual Release Process

If you prefer to create releases manually:

1. **Create and push a tag:**
   ```bash
   git tag -a v1.2.0 -m "Release 1.2.0"
   git push origin v1.2.0
   ```

2. **Monitor the workflow:**
   - Go to the [Actions tab](https://github.com/deviantony/pctl/actions)
   - Watch the "Release" workflow run
   - The release will be created automatically when the workflow completes

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR** (1.0.0): Breaking changes
- **MINOR** (1.1.0): New features, backward compatible
- **PATCH** (1.1.1): Bug fixes, backward compatible

For pre-releases, use suffixes:
- `1.2.0-beta.1`
- `1.2.0-rc.1`
- `1.2.0-alpha.1`

## Script Options

The release script supports several options:

```bash
# Create a release
./scripts/release.sh 1.2.0

# Dry run (see what would happen)
./scripts/release.sh 1.2.0 --dry-run

# Pre-release
./scripts/release.sh 1.2.0-beta.1

# Show help
./scripts/release.sh --help
```

## Troubleshooting

### Tag Already Exists
If you get "Tag already exists", you can:
- Delete the local tag: `git tag -d v1.2.0`
- Delete the remote tag: `git push origin --delete v1.2.0`
- Or use a different version number

### Workflow Fails
If the GitHub Actions workflow fails:
1. Check the [Actions tab](https://github.com/deviantony/pctl/actions) for error details
2. Common issues:
   - Build failures (check Go version compatibility)
   - Permission issues (check repository settings)
   - Network issues (retry the workflow)

### Manual Release
If the automated process fails, you can create a release manually:
1. Go to [Releases](https://github.com/deviantony/pctl/releases)
2. Click "Create a new release"
3. Select the tag
4. Add release notes
5. Upload the binaries from your local `build/` directory

## Files Created

After a successful release, the following files will be available:
- `pctl-{version}-linux-amd64`
- `pctl-{version}-linux-arm64`
- `pctl-{version}-darwin-amd64`
- `pctl-{version}-darwin-arm64`
- `pctl-{version}-windows-amd64.exe`
- `checksums.txt` (SHA256 checksums for all binaries)
