#!/bin/bash

# pctl Release Helper Script
# This script helps you create a new release by creating and pushing a tag

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 <version> [options]"
    echo ""
    echo "Arguments:"
    echo "  version    Version number (e.g., 1.1.1, 2.0.0-beta.1)"
    echo ""
    echo "Options:"
    echo "  --dry-run  Show what would be done without actually doing it"
    echo "  --help     Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 1.1.1"
    echo "  $0 2.0.0-beta.1"
    echo "  $0 1.1.1 --dry-run"
    echo ""
    echo "This script will:"
    echo "  1. Check that you're on the main branch"
    echo "  2. Ensure working directory is clean"
    echo "  3. Create and push a tag (v<version>)"
    echo "  4. GitHub Actions will automatically build and release"
}

# Parse arguments
VERSION=""
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        -*)
            print_error "Unknown option $1"
            show_usage
            exit 1
            ;;
        *)
            if [ -z "$VERSION" ]; then
                VERSION="$1"
            else
                print_error "Multiple versions specified"
                show_usage
                exit 1
            fi
            shift
            ;;
    esac
done

# Check if version is provided
if [ -z "$VERSION" ]; then
    print_error "Version is required"
    show_usage
    exit 1
fi

# Validate version format (basic check)
if [[ ! $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    print_warning "Version format might be invalid: $VERSION"
    print_warning "Expected format: X.Y.Z or X.Y.Z-suffix (e.g., 1.1.1, 2.0.0-beta.1)"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

TAG_NAME="v$VERSION"

print_status "Preparing release $VERSION (tag: $TAG_NAME)"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository"
    exit 1
fi

# Check if we're on main/master branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" && "$CURRENT_BRANCH" != "master" ]]; then
    print_warning "Not on main/master branch (current: $CURRENT_BRANCH)"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if working directory is clean
if ! git diff-index --quiet HEAD --; then
    print_error "Working directory is not clean. Please commit or stash your changes."
    git status --short
    exit 1
fi

# Check if tag already exists
if git rev-parse "$TAG_NAME" > /dev/null 2>&1; then
    print_error "Tag $TAG_NAME already exists"
    exit 1
fi

# Check if remote exists
if ! git remote get-url origin > /dev/null 2>&1; then
    print_error "No remote 'origin' found"
    exit 1
fi

print_success "All checks passed!"

if [ "$DRY_RUN" = true ]; then
    print_status "DRY RUN - Would execute:"
    echo "  git tag -s -a $TAG_NAME -m \"Release $VERSION\""
    echo "  git push origin $TAG_NAME"
    echo ""
    print_status "After pushing the tag, GitHub Actions will automatically:"
    echo "  - Build binaries for all platforms"
    echo "  - Create a GitHub release"
    echo "  - Upload all binaries and checksums"
    echo "  - Generate release notes"
else
    # Update README with new version
    print_status "Updating README with version $VERSION..."
    if [ -f "README.md" ]; then
        # Update all version references in README.md
        sed -i "s/pctl_[0-9]\+\.[0-9]\+\.[0-9]\+_/pctl_${VERSION}_/g" README.md
        
        # Check if any changes were made
        if git diff --quiet README.md; then
            print_status "No README changes needed"
        else
            print_status "README updated with version $VERSION"
            git add README.md
            git commit -m "docs: update installation instructions to v$VERSION"
        fi
    else
        print_warning "README.md not found, skipping version update"
    fi
    
    # Create and push tag
    print_status "Creating tag $TAG_NAME..."
    git tag -s -a "$TAG_NAME" -m "Release $VERSION"
    
    print_status "Pushing tag to origin..."
    git push origin "$TAG_NAME"
    
    print_success "Tag $TAG_NAME pushed successfully!"
    print_status "GitHub Actions will now automatically build and create the release."
    print_status "You can monitor the progress at: https://github.com/$(git remote get-url origin | sed 's/.*github.com[:/]\([^/]*\/[^/]*\)\.git/\1/')/actions"
fi
