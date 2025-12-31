#!/bin/bash
# Update Homebrew formulas with version and checksums from release
# Usage: ./scripts/update-homebrew.sh <version>
# Example: ./scripts/update-homebrew.sh 1.0.0

set -e

VERSION="${1}"
if [ -z "${VERSION}" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "${SCRIPT_DIR}/.." && pwd )"
FORMULA_DIR="${PROJECT_ROOT}/Formula"
DIST_DIR="${PROJECT_ROOT}/dist"

# Check if checksums file exists
if [ ! -f "${DIST_DIR}/checksums.txt" ]; then
    echo "Error: ${DIST_DIR}/checksums.txt not found"
    echo "Run 'make release VERSION=${VERSION}' first"
    exit 1
fi

echo "Updating Homebrew formulas for version ${VERSION}..."

# Function to get SHA256 for a specific archive
get_sha256() {
    local archive_name="$1"
    grep "${archive_name}" "${DIST_DIR}/checksums.txt" | awk '{print $1}'
}

# Update each formula
for binary in gatekey gatekey-server gatekey-gateway gatekey-admin; do
    formula_file="${FORMULA_DIR}/${binary}.rb"

    if [ ! -f "${formula_file}" ]; then
        echo "Warning: ${formula_file} not found, skipping"
        continue
    fi

    echo "Updating ${binary}.rb..."

    # Get checksums for each platform
    sha_darwin_amd64=$(get_sha256 "${binary}-${VERSION}-darwin-amd64.tar.gz")
    sha_darwin_arm64=$(get_sha256 "${binary}-${VERSION}-darwin-arm64.tar.gz")
    sha_linux_amd64=$(get_sha256 "${binary}-${VERSION}-linux-amd64.tar.gz")
    sha_linux_arm64=$(get_sha256 "${binary}-${VERSION}-linux-arm64.tar.gz")

    # Replace placeholders
    sed -i.bak \
        -e "s/VERSION_PLACEHOLDER/${VERSION}/g" \
        -e "s/SHA256_DARWIN_AMD64_PLACEHOLDER/${sha_darwin_amd64}/g" \
        -e "s/SHA256_DARWIN_ARM64_PLACEHOLDER/${sha_darwin_arm64}/g" \
        -e "s/SHA256_LINUX_AMD64_PLACEHOLDER/${sha_linux_amd64}/g" \
        -e "s/SHA256_LINUX_ARM64_PLACEHOLDER/${sha_linux_arm64}/g" \
        "${formula_file}"

    rm -f "${formula_file}.bak"

    echo "  - darwin/amd64: ${sha_darwin_amd64:0:16}..."
    echo "  - darwin/arm64: ${sha_darwin_arm64:0:16}..."
    echo "  - linux/amd64: ${sha_linux_amd64:0:16}..."
    echo "  - linux/arm64: ${sha_linux_arm64:0:16}..."
done

echo ""
echo "Formulas updated successfully!"
echo ""
echo "Next steps:"
echo "  1. Review changes in ${FORMULA_DIR}/"
echo "  2. Copy formulas to your Homebrew tap repository"
echo "  3. Commit and push the tap repository"
echo ""
echo "Example tap repository structure:"
echo "  homebrew-tap/"
echo "    Formula/"
echo "      gatekey.rb"
echo "      gatekey-server.rb"
echo "      gatekey-gateway.rb"
echo "      gatekey-admin.rb"
