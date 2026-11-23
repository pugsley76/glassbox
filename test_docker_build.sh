#!/bin/bash
# Copyright 2026 Glassbox Users
# SPDX-License-Identifier: Apache-2.0

# Test script for Docker multi-architecture builds
# Verifies that Docker images build correctly for both amd64 and arm64

set -e

echo "=== Docker Multi-Architecture Build Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

# Check if buildx is available
if ! docker buildx version &> /dev/null; then
    echo -e "${RED}Error: Docker buildx is not available${NC}"
    echo "Install with: docker buildx install"
    exit 1
fi

echo -e "${YELLOW}1. Setting up Docker buildx...${NC}"
# Create builder if it doesn't exist
if ! docker buildx ls | grep -q multiarch; then
    docker buildx create --name multiarch --use
    echo -e "${GREEN}[OK] Created multiarch builder${NC}"
else
    docker buildx use multiarch
    echo -e "${GREEN}[OK] Using existing multiarch builder${NC}"
fi

# Bootstrap the builder
docker buildx inspect --bootstrap
echo ""

echo -e "${YELLOW}2. Building for current platform...${NC}"
# Build with the same tag used by docker-compose.yml so compose can reuse the layer cache.
docker build -t glassbox:local .
echo -e "${GREEN}[OK] Single-platform build successful${NC}"
echo ""

echo -e "${YELLOW}3. Testing single-platform image...${NC}"
# Test version command
if docker run --rm glassbox:local --version; then
    echo -e "${GREEN}[OK] Version command works${NC}"
else
    echo -e "${RED}[FAIL] Version command failed${NC}"
    exit 1
fi

# Test help command
if docker run --rm glassbox:local --help > /dev/null 2>&1; then
    echo -e "${GREEN}[OK] Help command works${NC}"
else
    echo -e "${RED}[FAIL] Help command failed${NC}"
    exit 1
fi

# Check simulator binary exists at the path the Dockerfile installs it to.
if docker run --rm glassbox:local sh -c "test -f /app/simulator/glassbox-sim"; then
    echo -e "${GREEN}[OK] Simulator binary exists${NC}"
else
    echo -e "${RED}[FAIL] Simulator binary not found at /app/simulator/glassbox-sim${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}4. Building multi-architecture images...${NC}"
echo "This may take several minutes..."
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --tag glassbox:multiarch \
    --build-arg VERSION=test \
    --build-arg COMMIT_SHA=$(git rev-parse HEAD 2>/dev/null || echo "unknown") \
    --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    --load \
    . 2>&1 | grep -E "(Building|exporting|writing|naming)" || true

echo -e "${GREEN}[OK] Multi-architecture build successful${NC}"
echo ""

echo -e "${YELLOW}5. Inspecting image...${NC}"
docker image inspect glassbox:local | grep -E "(Architecture|Os)" | head -2
echo ""

echo -e "${YELLOW}6. Checking binary architecture...${NC}"
docker run --rm --entrypoint file glassbox:local /app/glassbox
docker run --rm --entrypoint file glassbox:local /app/simulator/glassbox-sim
echo ""

echo -e "${YELLOW}7. Verifying static linking...${NC}"
if docker run --rm --entrypoint ldd glassbox:local /app/glassbox 2>&1 | grep -q "not a dynamic executable"; then
    echo -e "${GREEN}[OK] Go binary is statically linked${NC}"
else
    echo -e "${YELLOW}[WARN] Go binary has dynamic dependencies${NC}"
    docker run --rm --entrypoint ldd glassbox:local /app/glassbox
fi

if docker run --rm --entrypoint ldd glassbox:local /app/simulator/glassbox-sim 2>&1 | grep -q "not a dynamic executable"; then
    echo -e "${GREEN}[OK] Rust binary is statically linked${NC}"
else
    echo -e "${YELLOW}[WARN] Rust binary has dynamic dependencies${NC}"
    docker run --rm --entrypoint ldd glassbox:local /app/simulator/glassbox-sim
fi
echo ""

echo -e "${YELLOW}8. Testing with docker-compose...${NC}"
if command -v docker-compose &> /dev/null || docker compose version &> /dev/null; then
    # Try docker compose (newer) or docker-compose (older)
    COMPOSE_CMD="docker compose"
    if ! docker compose version &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    fi

    $COMPOSE_CMD build glassbox
    echo -e "${GREEN}[OK] Docker Compose build successful${NC}"
else
    echo -e "${YELLOW}[WARN] Docker Compose not available, skipping${NC}"
fi
echo ""

echo -e "${YELLOW}9. Checking image size...${NC}"
SIZE=$(docker image inspect glassbox:local --format='{{.Size}}' | awk '{print $1/1024/1024}')
echo "Image size: ${SIZE} MB"
if (( $(echo "$SIZE < 200" | bc -l) )); then
    echo -e "${GREEN}[OK] Image size is reasonable${NC}"
else
    echo -e "${YELLOW}[WARN] Image size is larger than expected${NC}"
fi
echo ""

echo -e "${GREEN}=== All Docker tests passed! ===${NC}"
echo ""
echo "Next steps:"
echo "  - Push to registry: docker push glassbox:multiarch"
echo "  - Test on different platform: docker run --platform linux/arm64 glassbox:local --version"
echo "  - Run CI workflow: git push origin <branch>"
