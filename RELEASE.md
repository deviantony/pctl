# Release Process

This document explains how to create new releases for pctl using [GoReleaser](https://goreleaser.com/) and automated GitHub Actions.

## Quick Start

To create a new release, simply run:

```bash
./scripts/release.sh 1.1.1
```

This will:
1. Create a git tag `v1.1.1`
2. Push the tag to GitHub
3. Trigger the automated GoReleaser workflow
4. Build binaries for all platforms
5. Create a GitHub release with all binaries and release notes

## Prerequisites

- You must be on the `main` or `master` branch
- Your working directory must be clean (no uncommitted changes)
- You must have push access to the repository

## Release Workflow

The automated release process uses [GoReleaser](https://goreleaser.com/) and includes:

### 1. Build Process
GoReleaser builds binaries for multiple platforms:
- Linux AMD64 (`pctl_linux_amd64`)
- Linux ARM64 (`pctl_linux_arm64`)
- macOS AMD64 (`pctl_darwin_amd64`)
- macOS ARM64 (`pctl_darwin_arm64`)
- Windows AMD64 (`pctl_windows_amd64.exe`)

### 2. Release Creation
- Creates a GitHub release with the tag name
- Generates automatic release notes from git commits
- Uploads all built binaries and archives
- Creates and uploads SHA256 checksums

### 3. Release Notes
GoReleaser automatically generates release notes including:
- List of commits since the last release
- Installation instructions for all platforms
- Link to the full changelog

## Manual Release Process

If you prefer to create releases manually:

1. **Create and push a tag:**
   ```bash
   git tag -a v1.1.1 -m "Release 1.1.1"
   git push origin v1.1.1
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
- `1.1.1-beta.1`
- `1.1.1-rc.1`
- `1.1.1-alpha.1`

## Script Options

The release script supports several options:

```bash
# Create a release
./scripts/release.sh 1.1.1

# Dry run (see what would happen)
./scripts/release.sh 1.1.1 --dry-run

# Pre-release
./scripts/release.sh 1.1.1-beta.1

# Show help
./scripts/release.sh --help
```

## Troubleshooting

### Tag Already Exists
If you get "Tag already exists", you can:
- Delete the local tag: `git tag -d v1.1.1`
- Delete the remote tag: `git push origin --delete v1.1.1`
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
