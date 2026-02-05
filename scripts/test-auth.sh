#!/usr/bin/env bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL="${1:-http://localhost:8081}"
SERVICE_PID=""
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Function to cleanup on exit
cleanup() {
    if [ ! -z "$SERVICE_PID" ]; then
        echo -e "${YELLOW}Stopping auth-server (PID: $SERVICE_PID)${NC}"
        kill $SERVICE_PID 2>/dev/null
        wait $SERVICE_PID 2>/dev/null
    fi
}

trap cleanup EXIT

echo -e "${YELLOW}=== Go Auth Service Integration Test ===${NC}"
echo -e "Project root: $PROJECT_ROOT"
echo -e "Target URL: $BASE_URL\n"

# Step 1: Build binaries
echo -e "${YELLOW}1. Building binaries...${NC}"
cd "$PROJECT_ROOT"

go build -o bin/auth-server ./auth/cmd/server
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to build auth-server${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Built auth-server${NC}"

go build -o bin/test-puzzle-auth ./auth/cmd/test-puzzle-auth
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to build test-puzzle-auth${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Built test-puzzle-auth${NC}"

# Step 2: Start the auth server
echo -e "\n${YELLOW}2. Starting auth-server...${NC}"
./bin/auth-server &
SERVICE_PID=$!
echo -e "${GREEN}✓ Service started (PID: $SERVICE_PID)${NC}"

# Wait for service to start
echo -e "${YELLOW}   Waiting for service to be ready...${NC}"
for i in {1..10}; do
    if curl -s "$BASE_URL/auth/status" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Service is ready${NC}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}✗ Service failed to start within 10 seconds${NC}"
        exit 1
    fi
    sleep 1
done

# Step 3: Run the test utility
echo -e "\n${YELLOW}3. Running puzzle auth test...${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

./bin/test-puzzle-auth "$BASE_URL"
TEST_EXIT_CODE=$?

echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "\n${GREEN}=== Integration Test Passed ===${NC}"
else
    echo -e "\n${RED}=== Integration Test Failed (exit code: $TEST_EXIT_CODE) ===${NC}"
fi

exit $TEST_EXIT_CODE
