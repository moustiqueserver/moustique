#!/bin/bash
# Test all Moustique clients (HTTP + MQTT)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Get the project root (parent of tests/)
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to project root
cd "$PROJECT_ROOT"

# Default values
SERVER_IP="${1:-localhost}"
SERVER_PORT="${2:-33334}"
USERNAME="${3:-demo}"
PASSWORD="${4:-demo123}"
MQTT_PORT="${5:-1883}"

echo ""
echo "======================================================================"
echo "Testing all Moustique clients (HTTP + MQTT)"
echo "======================================================================"
echo "Server: $SERVER_IP:$SERVER_PORT"
echo "MQTT Port: $MQTT_PORT"
echo "Username: $USERNAME"
echo "======================================================================"
echo ""

FAILED_TESTS=()
PASSED_TESTS=()

# Function to run a test
run_test() {
    local name=$1
    local command=$2

    echo ""
    echo "----------------------------------------------------------------------"
    echo -e "${YELLOW}Running $name test...${NC}"
    echo "----------------------------------------------------------------------"

    if eval "$command"; then
        echo -e "${GREEN}✓ $name test PASSED${NC}"
        PASSED_TESTS+=("$name")
    else
        echo -e "${RED}✗ $name test FAILED${NC}"
        FAILED_TESTS+=("$name")
    fi
}

# Test Python client
if command -v python3 &> /dev/null; then
    run_test "Python" "cd '$PROJECT_ROOT/clients/python' && python3 tests/test_client.py $SERVER_IP $SERVER_PORT $USERNAME $PASSWORD $MQTT_PORT"
else
    echo -e "${YELLOW}⚠ Skipping Python test (python3 not found)${NC}"
fi

# Test JavaScript client
if command -v node &> /dev/null; then
    # Install dependencies if needed
    if [ ! -d "$PROJECT_ROOT/clients/javascript/node_modules" ]; then
        echo "Installing JavaScript dependencies..."
        (cd "$PROJECT_ROOT/clients/javascript" && npm install)
    fi
    run_test "JavaScript" "cd '$PROJECT_ROOT/clients/javascript' && node tests/test_client.js $SERVER_IP $SERVER_PORT $USERNAME $PASSWORD $MQTT_PORT"
else
    echo -e "${YELLOW}⚠ Skipping JavaScript test (node not found)${NC}"
fi

# Test Go client
if command -v go &> /dev/null; then
    # Get dependencies
    echo "Getting Go dependencies..."
    (cd "$PROJECT_ROOT/clients/go" && go mod tidy &> /dev/null)
    run_test "Go" "cd '$PROJECT_ROOT/clients/go' && go run test_client.go $SERVER_IP $SERVER_PORT $USERNAME $PASSWORD $MQTT_PORT"
else
    echo -e "${YELLOW}⚠ Skipping Go test (go not found)${NC}"
fi

# Test Java client
if command -v mvn &> /dev/null; then
    # Always compile to ensure latest changes are included
    echo "Compiling Java client..."
    (cd "$PROJECT_ROOT/clients/java" && mvn clean compile -q)

    run_test "Java" "cd '$PROJECT_ROOT/clients/java' && mvn exec:java -Dexec.mainClass=moustique.TestClient -Dexec.args=\"$SERVER_IP $SERVER_PORT $USERNAME $PASSWORD $MQTT_PORT\" -q"
else
    echo -e "${YELLOW}⚠ Skipping Java test (mvn not found)${NC}"
fi

# Test Perl client
if command -v perl &> /dev/null; then
    # Make test script executable
    chmod +x "$PROJECT_ROOT/clients/perl/test_client.pl"

    run_test "Perl" "cd '$PROJECT_ROOT/clients/perl' && perl test_client.pl $SERVER_IP $SERVER_PORT $USERNAME $PASSWORD $MQTT_PORT"
else
    echo -e "${YELLOW}⚠ Skipping Perl test (perl not found)${NC}"
fi

# Summary
echo ""
echo "======================================================================"
echo "TEST SUMMARY"
echo "======================================================================"
echo ""

if [ ${#PASSED_TESTS[@]} -gt 0 ]; then
    echo -e "${GREEN}PASSED (${#PASSED_TESTS[@]}):${NC}"
    for test in "${PASSED_TESTS[@]}"; do
        echo -e "  ${GREEN}✓${NC} $test"
    done
    echo ""
fi

if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    echo -e "${RED}FAILED (${#FAILED_TESTS[@]}):${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}✗${NC} $test"
    done
    echo ""
fi

TOTAL=$((${#PASSED_TESTS[@]} + ${#FAILED_TESTS[@]}))
echo "Total: ${#PASSED_TESTS[@]}/$TOTAL tests passed"
echo "======================================================================"
echo ""

# Exit with error if any tests failed
if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    exit 1
fi

exit 0
