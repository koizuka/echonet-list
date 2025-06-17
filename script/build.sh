#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building ECHONET Lite Controller...${NC}"

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Build server
echo -e "\n${GREEN}Building Go server...${NC}"
go build

# Build web UI
echo -e "\n${GREEN}Building Web UI...${NC}"
cd web
npm install
npm run build

echo -e "\n${GREEN}Build completed successfully!${NC}"
echo -e "Server binary: ${PROJECT_ROOT}/echonet-list"
echo -e "Web UI bundle: ${PROJECT_ROOT}/web/bundle/"