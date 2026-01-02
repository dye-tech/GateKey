#!/bin/bash
# GateKey Release Script
# Builds release binaries for all platforms and creates Homebrew-ready archives

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "${SCRIPT_DIR}/.." && pwd )"

cd "${PROJECT_ROOT}"

# Get version from git tag or argument
VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Output directory
DIST_DIR="${PROJECT_ROOT}/dist"
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

# Platforms to build for
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
)

# Binaries to build
BINARIES=(
    "gatekey:./cmd/gatekey"
    "gatekey-server:./cmd/gatekey-server"
    "gatekey-gateway:./cmd/gatekey-gateway"
    "gatekey-admin:./cmd/gatekey-admin"
    "gatekey-hub:./cmd/gatekey-hub"
    "gatekey-mesh-gateway:./cmd/gatekey-mesh-gateway"
)

# Build flags
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}"

echo -e "${GREEN}Building GateKey ${VERSION}${NC}"
echo "Commit: ${COMMIT}"
echo "Build Time: ${BUILD_TIME}"
echo ""

# Build each binary for each platform
for binary_spec in "${BINARIES[@]}"; do
    IFS=':' read -r binary_name binary_path <<< "${binary_spec}"

    for platform in "${PLATFORMS[@]}"; do
        IFS='/' read -r os arch <<< "${platform}"

        output_name="${binary_name}-${VERSION}-${os}-${arch}"
        output_dir="${DIST_DIR}/${output_name}"

        echo -e "${YELLOW}Building ${binary_name} for ${os}/${arch}...${NC}"

        mkdir -p "${output_dir}"

        # Build binary
        CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build \
            -ldflags "${LDFLAGS}" \
            -o "${output_dir}/${binary_name}" \
            "${binary_path}"

        # Copy documentation if available
        [ -f README.md ] && cp README.md "${output_dir}/"
        [ -f LICENSE ] && cp LICENSE "${output_dir}/"

        # Create archive
        tar -czf "${DIST_DIR}/${output_name}.tar.gz" -C "${DIST_DIR}" "${output_name}"

        # Clean up directory
        rm -rf "${output_dir}"

        echo -e "${GREEN}  Created ${output_name}.tar.gz${NC}"
    done
done

# Generate checksums
echo ""
echo -e "${YELLOW}Generating checksums...${NC}"
cd "${DIST_DIR}"
sha256sum *.tar.gz > checksums.txt
cd "${PROJECT_ROOT}"

echo ""
echo -e "${GREEN}Release build complete!${NC}"
echo "Archives created in: ${DIST_DIR}/"
echo ""
echo "Files:"
ls -la "${DIST_DIR}"/*.tar.gz
echo ""
echo "Checksums:"
cat "${DIST_DIR}/checksums.txt"
echo ""
echo -e "${YELLOW}To create a GitHub release:${NC}"
echo "  1. Tag the release: git tag v${VERSION}"
echo "  2. Push the tag: git push origin v${VERSION}"
echo "  3. Create release on GitHub and upload files from ${DIST_DIR}/"
echo ""
echo "Or use GitHub CLI:"
echo "  gh release create v${VERSION} ${DIST_DIR}/*.tar.gz ${DIST_DIR}/checksums.txt --title 'v${VERSION}' --notes 'Release v${VERSION}'"
