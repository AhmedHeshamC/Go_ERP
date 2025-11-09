#!/bin/bash

# ERPGo Docker Build Script
# This script builds optimized Docker images for production deployment

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Get version information
get_version_info() {
    local version="${APP_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
    local commit="${GIT_COMMIT:-$(git rev-parse HEAD 2>/dev/null || echo 'unknown')}"
    local build_time="${BUILD_TIME:-$(date -u '+%Y-%m-%d_%H:%M:%S UTC')}"

    echo "$version" "$commit" "$build_time"
}

# Build Docker image with optimization
build_image() {
    local service="$1"
    local dockerfile="$2"
    local image_name="$3"
    local tag="${4:-latest}"

    log "Building Docker image for $service: $image_name:$tag"

    # Check if Dockerfile exists
    if [[ ! -f "$PROJECT_ROOT/$dockerfile" ]]; then
        error "Dockerfile not found: $dockerfile"
        exit 1
    fi

    # Get version information
    local version_info
    read -r version commit build_time < <(get_version_info)

    # Build arguments
    local build_args=(
        "--build-arg" "APP_VERSION=$version"
        "--build-arg" "BUILD_TIME=$build_time"
        "--build-arg" "GIT_COMMIT=$commit"
        "--platform" "linux/amd64"
        "--no-cache"
        "--pull"
    )

    # Build the image
    log "Building with version: $version, commit: $commit, built: $build_time"

    if docker build \
        "${build_args[@]}" \
        -f "$dockerfile" \
        -t "$image_name:$tag" \
        .; then
        success "Successfully built $image_name:$tag"

        # Also tag with version if different from latest
        if [[ "$tag" != "$version" ]]; then
            docker tag "$image_name:$tag" "$image_name:$version"
            log "Also tagged as $image_name:$version"
        fi

        # Show image size
        local image_size=$(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" "$image_name:$tag" | tail -n1 | cut -f2)
        log "Image size: $image_size"

    else
        error "Failed to build $image_name:$tag"
        exit 1
    fi
}

# Push Docker image to registry
push_image() {
    local image_name="$1"
    local tag="${2:-latest}"

    log "Pushing Docker image: $image_name:$tag"

    if docker push "$image_name:$tag"; then
        success "Successfully pushed $image_name:$tag"

        # Also push version tag if different
        local version_info
        read -r version _ _ < <(get_version_info)
        if [[ "$tag" != "$version" ]]; then
            docker push "$image_name:$version" || warn "Failed to push version tag: $image_name:$version"
        fi
    else
        error "Failed to push $image_name:$tag"
        exit 1
    fi
}

# Scan Docker image for security vulnerabilities
scan_image() {
    local image_name="$1"
    local tag="${2:-latest}"

    log "Scanning Docker image for vulnerabilities: $image_name:$tag"

    # Check if trivy is available
    if ! command -v trivy &> /dev/null; then
        warn "Trivy not found, skipping security scan"
        return 0
    fi

    # Scan the image
    if trivy image --severity HIGH,CRITICAL --exit-code 1 "$image_name:$tag"; then
        success "Security scan passed for $image_name:$tag"
    else
        error "Security scan found vulnerabilities in $image_name:$tag"
        exit 1
    fi
}

# Multi-stage optimization check
optimize_image() {
    local image_name="$1"
    local tag="${2:-latest}"

    log "Analyzing image optimization for $image_name:$tag"

    # Get image details
    local image_id=$(docker images --format "{{.ID}}" "$image_name:$tag")
    local image_size=$(docker images --format "{{.Size}}" "$image_name:$tag")

    # Check layers
    local layer_count=$(docker history "$image_name:$tag" --format "{{.ID}}" | wc -l)

    log "Image ID: $image_id"
    log "Image size: $image_size"
    log "Layer count: $layer_count"

    # Recommendations for optimization
    if [[ $layer_count -gt 20 ]]; then
        warn "Consider reducing the number of layers (current: $layer_count)"
    fi

    # Check for unnecessary packages
    if docker run --rm "$image_name:$tag" sh -c "command -v gcc" &>/dev/null; then
        warn "Development tools found in final image - consider removing"
    fi

    success "Image optimization analysis completed"
}

# Clean up old images
cleanup_images() {
    local image_name="$1"
    local keep_count="${2:-3}"

    log "Cleaning up old images for $image_name (keeping last $keep_count)"

    # Remove old images except the last N
    local old_images=$(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.CreatedAt}}" "$image_name" | \
        grep -v "latest" | \
        sort -k2 -r | \
        tail -n +$((keep_count + 1)) | \
        awk '{print $1}')

    if [[ -n "$old_images" ]]; then
        echo "$old_images" | xargs -r docker rmi -f || true
        success "Cleaned up old images"
    else
        log "No old images to clean up"
    fi
}

# Main function
main() {
    local service="${1:-all}"
    local push="${PUSH_IMAGES:-false}"
    local scan="${SCAN_IMAGES:-false}"
    local registry="${DOCKER_REGISTRY:-}"

    log "Starting ERPGo Docker build process..."
    log "Service: $service"
    log "Push images: $push"
    log "Security scan: $scan"
    log "Registry: ${registry:-local}"

    # Change to project root
    cd "$PROJECT_ROOT"

    # Determine image name
    local base_image_name="${registry}erpgo"

    case "$service" in
        "api")
            build_image "api" "Dockerfile" "$base_image_name" "latest"
            ;;
        "worker")
            build_image "worker" "Dockerfile.worker" "$base_image_name-worker" "latest"
            ;;
        "migrator")
            build_image "migrator" "Dockerfile.migrator" "$base_image_name-migrator" "latest"
            ;;
        "all")
            # Build all services
            build_image "api" "Dockerfile" "$base_image_name" "latest"
            build_image "worker" "Dockerfile.worker" "$base_image_name-worker" "latest"
            build_image "migrator" "Dockerfile.migrator" "$base_image_name-migrator" "latest"
            ;;
        *)
            error "Unknown service: $service"
            echo "Available services: api, worker, migrator, all"
            exit 1
            ;;
    esac

    # Optimization analysis
    if [[ "$service" == "all" ]]; then
        optimize_image "$base_image_name" "latest"
        optimize_image "$base_image_name-worker" "latest"
        optimize_image "$base_image_name-migrator" "latest"
    else
        local image_name="$base_image_name"
        if [[ "$service" != "api" ]]; then
            image_name="$base_image_name-$service"
        fi
        optimize_image "$image_name" "latest"
    fi

    # Security scanning
    if [[ "$scan" == "true" ]]; then
        log "Starting security scans..."

        if [[ "$service" == "all" ]]; then
            scan_image "$base_image_name" "latest"
            scan_image "$base_image_name-worker" "latest"
            scan_image "$base_image_name-migrator" "latest"
        else
            local image_name="$base_image_name"
            if [[ "$service" != "api" ]]; then
                image_name="$base_image_name-$service"
            fi
            scan_image "$image_name" "latest"
        fi
    fi

    # Push images
    if [[ "$push" == "true" ]]; then
        log "Pushing images to registry..."

        if [[ "$service" == "all" ]]; then
            push_image "$base_image_name" "latest"
            push_image "$base_image_name-worker" "latest"
            push_image "$base_image_name-migrator" "latest"
        else
            local image_name="$base_image_name"
            if [[ "$service" != "api" ]]; then
                image_name="$base_image_name-$service"
            fi
            push_image "$image_name" "latest"
        fi
    fi

    success "Docker build process completed successfully!"
}

# Show usage
usage() {
    echo "ERPGo Docker Build Script"
    echo ""
    echo "Usage:"
    echo "  $0 [SERVICE] [OPTIONS]"
    echo ""
    echo "Services:"
    echo "  api         Build API service image"
    echo "  worker      Build worker service image"
    echo "  migrator    Build migrator service image"
    echo "  all         Build all images (default)"
    echo ""
    echo "Environment Variables:"
    echo "  APP_VERSION    Application version (default: git describe)"
    echo "  GIT_COMMIT     Git commit hash (default: git rev-parse)"
    echo "  BUILD_TIME     Build timestamp (default: current time)"
    echo "  PUSH_IMAGES    Push images to registry (default: false)"
    echo "  SCAN_IMAGES    Run security scans (default: false)"
    echo "  DOCKER_REGISTRY Registry prefix (default: none)"
    echo ""
    echo "Examples:"
    echo "  $0 api                    # Build API image"
    echo "  $0 all                    # Build all images"
    echo "  PUSH_IMAGES=true $0 all   # Build and push all images"
    echo "  SCAN_IMAGES=true $0 api   # Build and scan API image"
    echo "  DOCKER_REGISTRY=myreg.com/ $0 all  # Build with custom registry"
}

# Parse command line arguments
case "${1:-}" in
    "help"|"-h"|"--help")
        usage
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac