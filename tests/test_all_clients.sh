#!/bin/bash
# Test suite for all Moustique clients
# Tests: Python, JavaScript, Java, Go, Perl, and CLI

# set -e  # Exit on error - commented out for debugging

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
MOUSTIQUE_HOST="${MOUSTIQUE_HOST:-localhost}"
MOUSTIQUE_PORT="${MOUSTIQUE_PORT:-33334}"
TEST_USER="${TEST_USER:-testuser}"
TEST_PASS="${TEST_PASS:-testpass123}"
TEST_TOPIC="/test/client"

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print colored output
print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    ((TESTS_PASSED++))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Function to run a test
run_test() {
    ((TESTS_RUN++))
    local test_name="$1"
    local test_cmd="$2"

    echo ""
    echo -e "${YELLOW}Testing: $test_name${NC}"

    if eval "$test_cmd" > /tmp/test_output.log 2>&1; then
        print_success "$test_name passed"
        return 0
    else
        print_error "$test_name failed"
        echo "Command: $test_cmd"
        echo "Output:"
        cat /tmp/test_output.log
        return 1
    fi
}

# Check if server is running
check_server() {
    print_info "Checking if Moustique server is running on $MOUSTIQUE_HOST:$MOUSTIQUE_PORT..."
    if curl -s "http://$MOUSTIQUE_HOST:$MOUSTIQUE_PORT/VERSION" > /dev/null 2>&1; then
        print_success "Server is running"
        return 0
    else
        print_error "Server is not running on $MOUSTIQUE_HOST:$MOUSTIQUE_PORT"
        echo "Please start the server first: ./moustique"
        exit 1
    fi
}

# Main test execution
print_header "Moustique Client Test Suite"
echo "Host: $MOUSTIQUE_HOST"
echo "Port: $MOUSTIQUE_PORT"
echo "Test User: $TEST_USER"
echo ""

check_server

# ============================================================================
# CLI Tests
# ============================================================================
print_header "Testing CLI Client"

if [ -f "./moustique-cli" ]; then
    run_test "CLI: Publish to public broker" \
        "./moustique-cli -h $MOUSTIQUE_HOST -p $MOUSTIQUE_PORT -a pub -t $TEST_TOPIC/cli -m 'Test from CLI'"

    run_test "CLI: Publish with authentication" \
        "./moustique-cli -h $MOUSTIQUE_HOST -p $MOUSTIQUE_PORT -u $TEST_USER -pwd $TEST_PASS -a pub -t $TEST_TOPIC/cli/auth -m 'Auth test'"

    run_test "CLI: Put value" \
        "./moustique-cli -h $MOUSTIQUE_HOST -p $MOUSTIQUE_PORT -a put -t $TEST_TOPIC/cli/value -m 'stored_value'"

    run_test "CLI: Version" \
        "./moustique-cli -a version"
else
    print_error "CLI client not found (./moustique-cli). Run 'make cli' to build it."
fi

# ============================================================================
# Python Tests
# ============================================================================
print_header "Testing Python Client"

if command -v python3 &> /dev/null; then
    run_test "Python: Public broker publish" \
        "python3 tests/python_test.py public"

    run_test "Python: Authenticated publish" \
        "python3 tests/python_test.py auth $TEST_USER $TEST_PASS"

    run_test "Python: PUTVAL" \
        "python3 tests/python_test.py putval $TEST_USER $TEST_PASS"

    run_test "Python: GETVAL" \
        "python3 tests/python_test.py getval $TEST_USER $TEST_PASS"

    run_test "Python: SUBSCRIBE/PICKUP" \
        "python3 tests/python_test.py subscribe $TEST_USER $TEST_PASS"
else
    print_error "Python3 not found. Skipping Python tests."
fi

# ============================================================================
# JavaScript Tests
# ============================================================================
print_header "Testing JavaScript Client"

if command -v node &> /dev/null; then
    run_test "Node.js: Public broker publish" \
        "node tests/javascript_test.mjs public"

    run_test "Node.js: Authenticated publish" \
        "node tests/javascript_test.mjs auth $TEST_USER $TEST_PASS"

    run_test "Node.js: PUTVAL" \
        "node tests/javascript_test.mjs putval $TEST_USER $TEST_PASS"

    run_test "Node.js: GETVAL" \
        "node tests/javascript_test.mjs getval $TEST_USER $TEST_PASS"

    run_test "Node.js: SUBSCRIBE/PICKUP" \
        "node tests/javascript_test.mjs subscribe $TEST_USER $TEST_PASS"
else
    print_error "Node.js not found. Skipping JavaScript tests."
fi

# ============================================================================
# Go Tests
# ============================================================================
print_header "Testing Go Client"

run_test "Go: Public broker publish" \
    "(cd tests && go run test_go_client.go public $MOUSTIQUE_HOST $MOUSTIQUE_PORT)"

run_test "Go: Authenticated publish" \
    "(cd tests && go run test_go_client.go auth $MOUSTIQUE_HOST $MOUSTIQUE_PORT $TEST_USER $TEST_PASS)"

run_test "Go: PUTVAL" \
    "(cd tests && go run test_go_client.go putval $MOUSTIQUE_HOST $MOUSTIQUE_PORT $TEST_USER $TEST_PASS)"

run_test "Go: SUBSCRIBE/PICKUP" \
    "(cd tests && go run test_go_client.go subscribe $MOUSTIQUE_HOST $MOUSTIQUE_PORT $TEST_USER $TEST_PASS)"

# ============================================================================
# Perl Tests
# ============================================================================
print_header "Testing Perl Client"

if command -v perl &> /dev/null; then
    run_test "Perl: Public broker publish" \
        "perl tests/perl_test.pl public $MOUSTIQUE_HOST $MOUSTIQUE_PORT"

    run_test "Perl: Authenticated publish" \
        "perl tests/perl_test.pl auth $MOUSTIQUE_HOST $MOUSTIQUE_PORT $TEST_USER $TEST_PASS"
else
    print_error "Perl not found. Skipping Perl tests."
fi

# ============================================================================
# Java Tests (optional - requires Maven/Gradle)
# ============================================================================
print_header "Testing Java Client"

if command -v javac &> /dev/null && [ -f "clients/java/src/main/java/moustique/MoustiqueClient.java" ]; then
    print_info "Java client found but tests require Maven/Gradle setup. Skipping for now."
    # TODO: Add Java tests when build system is set up
else
    print_info "Java not configured. Skipping Java tests."
fi

# ============================================================================
# Summary
# ============================================================================
print_header "Test Summary"

echo "Total tests run: $TESTS_RUN"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    print_success "All tests passed! 🎉"
    exit 0
else
    echo ""
    print_error "Some tests failed. Please check the output above."
    exit 1
fi
