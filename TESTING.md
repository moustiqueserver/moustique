# Testing Guide for Moustique

Quick guide to testing all Moustique components.

## Quick Start

```bash
# 1. Build everything
make all

# 2. Start server (in terminal 1)
./moustique

# 3. Create test user via superadmin UI
# Open: http://localhost:33334/superadmin
# Create user: testuser / testpass123

# 4. Run all client tests (in terminal 2)
make test-clients
```

## Test Components

### 1. Server Unit Tests (Go)

```bash
make test
```

### 2. Client Integration Tests

Tests all clients (CLI, Python, JavaScript, Go, Perl) against running server.

```bash
# Start server first
./moustique

# Run all client tests
make test-clients

# Or run individually
./tests/test_all_clients.sh
```

### 3. Manual Testing

#### CLI Client
```bash
# Public broker
./moustique-cli -a pub -t /test -m "hello"

# With auth
./moustique-cli -u testuser -pwd testpass123 -a pub -t /test -m "hello"

# Subscribe
./moustique-cli -a sub -t /test/#
```

#### Python Client
```bash
python3 tests/python_test.py public
python3 tests/python_test.py auth testuser testpass123
```

#### JavaScript Client
```bash
node tests/javascript_test.js public
node tests/javascript_test.js auth testuser testpass123
```

#### Go Client
```bash
cd tests && go build -o go_test go_test.go
./go_test public localhost 33334
./go_test auth localhost 33334 testuser testpass123
```

#### Perl Client
```bash
perl tests/perl_test.pl public localhost 33334
perl tests/perl_test.pl auth localhost 33334 testuser testpass123
```

## Configuration

Set environment variables to customize tests:

```bash
export MOUSTIQUE_HOST="localhost"
export MOUSTIQUE_PORT="33334"
export TEST_USER="testuser"
export TEST_PASS="testpass123"

./tests/test_all_clients.sh
```

## Test Results

The test suite will show:
- ✅ Green checkmarks for passing tests
- ❌ Red X marks for failing tests
- Summary at the end with pass/fail counts

Example output:
```
================================================
Testing CLI Client
================================================

Testing: CLI: Publish to public broker
✓ CLI: Publish to public broker passed

Testing: CLI: Publish with authentication
✓ CLI: Publish with authentication passed

================================================
Test Summary
================================================
Total tests run: 15
Passed: 15
Failed: 0

✓ All tests passed! 🎉
```

## Troubleshooting

### Server Not Running
```
✗ Server is not running on localhost:33334
```
**Fix:** Start the server: `./moustique`

### Authentication Failed
```
✗ Python authenticated publish failed: Invalid credentials
```
**Fix:** Create test user via superadmin UI or adjust `TEST_USER`/`TEST_PASS`

### Client Not Found
```
✗ CLI client not found (./moustique-cli)
```
**Fix:** Build the client: `make cli`

## Continuous Integration

For CI/CD pipelines:

```bash
#!/bin/bash
# ci-test.sh

# Build
make all

# Start server in background
./moustique &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Run tests
make test-clients
TEST_RESULT=$?

# Stop server
kill $SERVER_PID

# Exit with test result
exit $TEST_RESULT
```

## Adding New Tests

1. Create test file in `tests/` directory
2. Follow pattern: accept `public` or `auth` mode
3. Return exit code 0 on success, 1 on failure
4. Add to `test_all_clients.sh`

See [tests/README.md](tests/README.md) for details.

## Before Committing

Always run the full test suite before committing:

```bash
# Clean build
make clean
make all

# Run tests
make test           # Unit tests
make test-clients  # Integration tests (requires running server)
```

## Need Help?

- See [tests/README.md](tests/README.md) for detailed documentation
- Check test output for specific error messages
- Ensure server is running and accessible
- Verify test user exists with correct credentials
