#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Parse arguments
TARGET="${1:-all}"

case "$TARGET" in
    server|web|web-ui|all)
        ;;
    *)
        echo -e "${RED}Usage: $0 [server|web|web-ui|all]${NC}"
        echo "  server  - Build Go server only"
        echo "  web     - Build Web UI only"
        echo "  web-ui  - Build Web UI only (alias for web)"
        echo "  all     - Build both (default)"
        exit 1
        ;;
esac

echo -e "${YELLOW}Building ECHONET Lite Controller ($TARGET)...${NC}"

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Build server
if [[ "$TARGET" == "server" || "$TARGET" == "all" ]]; then
    echo -e "\n${GREEN}Building Go server...${NC}"
    go build
    echo -e "Server binary: ${PROJECT_ROOT}/echonet-list"
fi

# Build web UI
if [[ "$TARGET" == "web" || "$TARGET" == "web-ui" || "$TARGET" == "all" ]]; then
    echo -e "\n${GREEN}Building Web UI...${NC}"
    cd web
    npm install
    npm run build
    echo -e "Web UI bundle: ${PROJECT_ROOT}/web/bundle/"
fi

echo -e "\n${GREEN}Build completed successfully!${NC}"