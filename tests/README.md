# Moustique Client Test Suite

Automated tests for all Moustique client libraries.

## Overview

This test suite validates that all client implementations (Python, JavaScript, Go, Perl, CLI) can successfully:
- Connect to the Moustique server
- Publish messages to public broker
- Publish messages with authentication
- Use correct encoding (ROT13 + Base64)

## Prerequisites

1. **Running Moustique server** on localhost:33334 (or configure via environment variables)
2. **Test user created** with username `testuser` and password `testpass123` (or configure via environment variables)

### Creating Test User

```bash
# Start server
./moustique

# In another terminal, use superadmin UI or CLI to create user
# Or via API (example with curl):
# First, create user via superadmin interface at http://localhost:33334/superadmin
```

## Running Tests

### Quick Start - Run All Tests

```bash
# Make test script executable
chmod +x tests/test_all_clients.sh

# Run all tests
./tests/test_all_clients.sh
```

### With Custom Configuration

```bash
# Set environment variables
export MOUSTIQUE_HOST="moustique.example.com"
export MOUSTIQUE_PORT="33334"
export TEST_USER="myuser"
export TEST_PASS="mypassword"

# Run tests
./tests/test_all_clients.sh
```

### Individual Client Tests

```bash
# Python
python3 tests/python_test.py public
python3 tests/python_test.py auth testuser testpass123

# JavaScript/Node.js
node tests/javascript_test.js public
node tests/javascript_test.js auth testuser testpass123

# Go
cd tests && go build -o go_test go_test.go
./go_test public localhost 33334
./go_test auth localhost 33334 testuser testpass123

# Perl
perl tests/perl_test.pl public localhost 33334
perl tests/perl_test.pl auth localhost 33334 testuser testpass123

# CLI
./moustique-cli -a pub -t /test/cli -m "test"
./moustique-cli -u testuser -pwd testpass123 -a pub -t /test/cli/auth -m "test"
```

## Test Structure

```
tests/
├── README.md                 # This file
├── test_all_clients.sh      # Main test runner
├── python_test.py           # Python client test
├── javascript_test.js       # JavaScript/Node.js client test
├── go_test.go              # Go client test
└── perl_test.pl            # Perl client test
```

## Adding New Tests

To add a new test:

1. Create test file in `tests/` directory
2. Implement test modes: `public` and `auth`
3. Return exit code 0 on success, 1 on failure
4. Add test to `test_all_clients.sh`

### Example Test Template

```bash
#!/usr/bin/env your-language

# Parse arguments
mode = args[0]
host = args[1]
port = args[2]

if mode == "public":
    # Test without authentication
    client = create_client(host, port)
    client.publish("/test/topic", "message")
    exit(0)

elif mode == "auth":
    username = args[3]
    password = args[4]
    # Test with authentication
    client = create_client(host, port, username, password)
    client.publish("/test/topic", "message")
    exit(0)

exit(1)  # Failed
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Client Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Build Moustique server
        run: make server

      - name: Start server
        run: ./moustique &

      - name: Wait for server
        run: sleep 2

      - name: Create test user
        run: |
          # TODO: Add user creation via API

      - name: Run tests
        run: ./tests/test_all_clients.sh
```

## Troubleshooting

### Server Not Running

```
✗ Server is not running on localhost:33334
Please start the server first: ./moustique
```

**Solution:** Start the Moustique server before running tests.

### Authentication Failed

```
✗ Python authenticated publish failed: Authentication failed: Invalid username or password
```

**Solution:** Create test user with correct credentials:
- Username: `testuser` (or set `TEST_USER`)
- Password: `testpass123` (or set `TEST_PASS`)

### Client Not Found

```
✗ CLI client not found (./moustique-cli). Run 'make cli' to build it.
```

**Solution:** Build missing client:
```bash
make cli           # For CLI
make all          # For server and CLI
pip install -e clients/python   # For Python
cd clients/javascript && npm install  # For JavaScript
```

### Port Already in Use

If the default port 33334 is in use:

```bash
# Start server on different port
./moustique -port 33335

# Run tests with custom port
export MOUSTIQUE_PORT="33335"
./tests/test_all_clients.sh
```

## Test Coverage

Current test coverage:

- ✅ CLI client
  - Public broker publish
  - Authenticated publish
  - Put value
  - Version check

- ✅ Python client
  - Public broker publish
  - Authenticated publish

- ✅ JavaScript client
  - Public broker publish
  - Authenticated publish

- ✅ Go client
  - Public broker publish
  - Authenticated publish

- ✅ Perl client
  - Public broker publish
  - Authenticated publish

- ⚠️ Java client
  - Requires Maven/Gradle setup (TODO)

## Future Improvements

- [ ] Add subscribe/pickup tests
- [ ] Add value storage (putval/getval) tests
- [ ] Add wildcard subscription tests
- [ ] Add performance/load tests
- [ ] Add integration tests with multiple clients
- [ ] Add Docker-based test environment
- [ ] Add Java client tests
- [ ] Add automated user creation in tests
- [ ] Add test for server restart/reconnection

## Contributing

When adding a new client or modifying existing clients:

1. Update or create corresponding test file
2. Run full test suite before committing
3. Ensure all tests pass
4. Update this README if adding new test types

```bash
# Before committing
./tests/test_all_clients.sh

# If all tests pass, you're good to go!
```
