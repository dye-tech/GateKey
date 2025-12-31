#!/bin/bash
# Build and push GateKey images to Harbor registry
set -e

# Configuration
REGISTRY="${REGISTRY:-harbor.dye.tech}"
PROJECT="${PROJECT:-library}"
VERSION="${VERSION:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building GateKey images...${NC}"

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "${SCRIPT_DIR}/.." && pwd )"

cd "${PROJECT_ROOT}"

# Build server image
echo -e "${YELLOW}Building gatekey-server...${NC}"
docker build -t "${REGISTRY}/${PROJECT}/gatekey-server:${VERSION}" -f Dockerfile .

# Build gateway image
echo -e "${YELLOW}Building gatekey-gateway...${NC}"
docker build -t "${REGISTRY}/${PROJECT}/gatekey-gateway:${VERSION}" -f Dockerfile.gateway .

# Build web image (if web directory exists and has package.json)
if [ -f "web/package.json" ]; then
    echo -e "${YELLOW}Building gatekey-web...${NC}"
    docker build -t "${REGISTRY}/${PROJECT}/gatekey-web:${VERSION}" -f Dockerfile.web .
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

if [ -f "web/package.json" ]; then
    docker push "${REGISTRY}/${PROJECT}/gatekey-web:${VERSION}"
    echo -e "${GREEN}Pushed gatekey-web:${VERSION}${NC}"
fi

echo -e "${GREEN}All images pushed successfully!${NC}"
echo ""
echo "Images:"
echo "  - ${REGISTRY}/${PROJECT}/gatekey-server:${VERSION}"
echo "  - ${REGISTRY}/${PROJECT}/gatekey-gateway:${VERSION}"
if [ -f "web/package.json" ]; then
    echo "  - ${REGISTRY}/${PROJECT}/gatekey-web:${VERSION}"
fi
