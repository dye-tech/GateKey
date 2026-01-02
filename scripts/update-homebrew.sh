#!/bin/bash
# Update Homebrew formulas with new version and checksums
# Usage: ./update-homebrew.sh <version> <homebrew-tap-path>

set -e

VERSION="${1:-}"
HOMEBREW_TAP="${2:-/home/jesse/Desktop/homebrew-gatekey}"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version> [homebrew-tap-path]"
    echo "Example: $0 1.2.0"
    exit 1
fi

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

DIST_DIR="$(dirname "$0")/../dist"

if [[ ! -d "$DIST_DIR" ]]; then
    echo "Error: dist directory not found. Run release.sh first."
    exit 1
fi

echo "Updating Homebrew formulas to version ${VERSION}..."

# Formulas and their corresponding binary names
FORMULAS="gatekey gatekey-admin gatekey-server gatekey-gateway gatekey-hub gatekey-mesh-gateway"

for formula in $FORMULAS; do
    binary="$formula"
    formula_file="${HOMEBREW_TAP}/Formula/${formula}.rb"

    if [[ ! -f "$formula_file" ]]; then
        echo "Warning: Formula file not found: $formula_file"
        continue
    fi

    echo "Updating ${formula}..."

    # Get checksums from dist directory
    darwin_arm64_sha=$(sha256sum "${DIST_DIR}/${binary}-${VERSION}-darwin-arm64.tar.gz" 2>/dev/null | cut -d' ' -f1 || echo "MISSING")
    darwin_amd64_sha=$(sha256sum "${DIST_DIR}/${binary}-${VERSION}-darwin-amd64.tar.gz" 2>/dev/null | cut -d' ' -f1 || echo "MISSING")
    linux_arm64_sha=$(sha256sum "${DIST_DIR}/${binary}-${VERSION}-linux-arm64.tar.gz" 2>/dev/null | cut -d' ' -f1 || echo "MISSING")
    linux_amd64_sha=$(sha256sum "${DIST_DIR}/${binary}-${VERSION}-linux-amd64.tar.gz" 2>/dev/null | cut -d' ' -f1 || echo "MISSING")

    if [[ "$darwin_arm64_sha" == "MISSING" ]]; then
        echo "  Warning: Could not find archives for ${binary}"
        continue
    fi

    # Update version
    sed -i "s/version \"[^\"]*\"/version \"${VERSION}\"/" "$formula_file"

    # Update checksums using awk
    awk -v darwin_arm64="$darwin_arm64_sha" \
        -v darwin_amd64="$darwin_amd64_sha" \
        -v linux_arm64="$linux_arm64_sha" \
        -v linux_amd64="$linux_amd64_sha" '
    BEGIN { sha_count = 0; in_macos = 0; in_linux = 0 }
    /on_macos do/ { in_macos = 1; in_linux = 0 }
    /on_linux do/ { in_linux = 1; in_macos = 0 }
    /sha256 "/ {
        if (in_macos && sha_count == 0) {
            sub(/sha256 "[^"]*"/, "sha256 \"" darwin_arm64 "\"")
            sha_count++
        } else if (in_macos && sha_count == 1) {
            sub(/sha256 "[^"]*"/, "sha256 \"" darwin_amd64 "\"")
            sha_count++
        } else if (in_linux && sha_count == 2) {
            sub(/sha256 "[^"]*"/, "sha256 \"" linux_arm64 "\"")
            sha_count++
        } else if (in_linux && sha_count == 3) {
            sub(/sha256 "[^"]*"/, "sha256 \"" linux_amd64 "\"")
            sha_count++
        }
    }
    { print }
    ' "$formula_file" > "${formula_file}.tmp" && mv "${formula_file}.tmp" "$formula_file"

    echo "  Updated: darwin-arm64=${darwin_arm64_sha:0:8}... darwin-amd64=${darwin_amd64_sha:0:8}..."
done

echo ""
echo "Done! Now commit and push the homebrew tap:"
echo "  cd ${HOMEBREW_TAP}"
echo "  git add -A"
echo "  git commit -m 'Update to v${VERSION}'"
echo "  git push"
