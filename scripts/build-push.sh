#!/bin/bash
# Build and push GateKey images to Harbor registry
set -e

# Configuration
REGISTRY="${REGISTRY:-harbor.dye.tech}"
PROJECT="${PROJECT:-library}"
VERSION="${VERSION:-latest}"
NO_CACHE="${NO_CACHE:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build args - use --no-cache by default to ensure fresh builds
BUILD_ARGS=""
if [ "${NO_CACHE}" = "true" ]; then
    BUILD_ARGS="--no-cache"
    echo -e "${YELLOW}Building with --no-cache (set NO_CACHE=false to use cache)${NC}"
fi

echo -e "${GREEN}Building GateKey images...${NC}"

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "${SCRIPT_DIR}/.." && pwd )"

cd "${PROJECT_ROOT}"

# Build server image
echo -e "${YELLOW}Building gatekey-server...${NC}"
docker build ${BUILD_ARGS} -t "${REGISTRY}/${PROJECT}/gatekey-server:${VERSION}" -f Dockerfile .

# Build gateway image (OpenVPN)
echo -e "${YELLOW}Building gatekey-gateway...${NC}"
docker build ${BUILD_ARGS} -t "${REGISTRY}/${PROJECT}/gatekey-gateway:${VERSION}" -f Dockerfile.gateway .

# Build hub image (OpenVPN mesh)
echo -e "${YELLOW}Building gatekey-hub...${NC}"
docker build ${BUILD_ARGS} -t "${REGISTRY}/${PROJECT}/gatekey-hub:${VERSION}" -f Dockerfile.hub .

# Build wireguard-gateway image
echo -e "${YELLOW}Building gatekey-wireguard-gateway...${NC}"
docker build ${BUILD_ARGS} -t "${REGISTRY}/${PROJECT}/gatekey-wireguard-gateway:${VERSION}" -f Dockerfile.wireguard-gateway .

# Build wireguard-hub image (if Dockerfile exists)
if [ -f "Dockerfile.wireguard-hub" ]; then
    echo -e "${YELLOW}Building gatekey-wireguard-hub...${NC}"
    docker build ${BUILD_ARGS} -t "${REGISTRY}/${PROJECT}/gatekey-wireguard-hub:${VERSION}" -f Dockerfile.wireguard-hub .
fi

# Build wireguard-mesh-gateway image (if Dockerfile exists)
if [ -f "Dockerfile.wireguard-mesh-gateway" ]; then
    echo -e "${YELLOW}Building gatekey-wireguard-mesh-gateway...${NC}"
    docker build ${BUILD_ARGS} -t "${REGISTRY}/${PROJECT}/gatekey-wireguard-mesh-gateway:${VERSION}" -f Dockerfile.wireguard-mesh-gateway .
fi

# Build web image (if web directory exists and has package.json)
if [ -f "web/package.json" ]; then
    echo -e "${YELLOW}Building gatekey-web...${NC}"
    docker build ${BUILD_ARGS} -t "${REGISTRY}/${PROJECT}/gatekey-web:${VERSION}" -f Dockerfile.web .
else
    echo -e "${YELLOW}Skipping gatekey-web (web/package.json not found)${NC}"
fi

echo -e "${GREEN}Build complete!${NC}"

# Push images
echo -e "${YELLOW}Pushing images to ${REGISTRY}...${NC}"

docker push "${REGISTRY}/${PROJECT}/gatekey-server:${VERSION}"
echo -e "${GREEN}Pushed gatekey-server:${VERSION}${NC}"

docker push "${REGISTRY}/${PROJECT}/gatekey-gateway:${VERSION}"
echo -e "${GREEN}Pushed gatekey-gateway:${VERSION}${NC}"

docker push "${REGISTRY}/${PROJECT}/gatekey-hub:${VERSION}"
echo -e "${GREEN}Pushed gatekey-hub:${VERSION}${NC}"

docker push "${REGISTRY}/${PROJECT}/gatekey-wireguard-gateway:${VERSION}"
echo -e "${GREEN}Pushed gatekey-wireguard-gateway:${VERSION}${NC}"

if [ -f "Dockerfile.wireguard-hub" ]; then
    docker push "${REGISTRY}/${PROJECT}/gatekey-wireguard-hub:${VERSION}"
    echo -e "${GREEN}Pushed gatekey-wireguard-hub:${VERSION}${NC}"
fi

if [ -f "Dockerfile.wireguard-mesh-gateway" ]; then
    docker push "${REGISTRY}/${PROJECT}/gatekey-wireguard-mesh-gateway:${VERSION}"
    echo -e "${GREEN}Pushed gatekey-wireguard-mesh-gateway:${VERSION}${NC}"
fi

if [ -f "web/package.json" ]; then
    docker push "${REGISTRY}/${PROJECT}/gatekey-web:${VERSION}"
    echo -e "${GREEN}Pushed gatekey-web:${VERSION}${NC}"
fi

echo -e "${GREEN}All images pushed successfully!${NC}"
echo ""
echo "Images:"
echo "  - ${REGISTRY}/${PROJECT}/gatekey-server:${VERSION}"
echo "  - ${REGISTRY}/${PROJECT}/gatekey-gateway:${VERSION}"
echo "  - ${REGISTRY}/${PROJECT}/gatekey-hub:${VERSION}"
echo "  - ${REGISTRY}/${PROJECT}/gatekey-wireguard-gateway:${VERSION}"
if [ -f "Dockerfile.wireguard-hub" ]; then
    echo "  - ${REGISTRY}/${PROJECT}/gatekey-wireguard-hub:${VERSION}"
fi
if [ -f "Dockerfile.wireguard-mesh-gateway" ]; then
    echo "  - ${REGISTRY}/${PROJECT}/gatekey-wireguard-mesh-gateway:${VERSION}"
fi
if [ -f "web/package.json" ]; then
    echo "  - ${REGISTRY}/${PROJECT}/gatekey-web:${VERSION}"
fi
