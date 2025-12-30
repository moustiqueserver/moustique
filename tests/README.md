# Moustique Client Tests

This directory contains integration tests for all Moustique client implementations.

## Overview

Each client test validates both **HTTP (polling)** and **MQTT (push)** protocols:

- **Python** (`clients/python/tests/test_client.py`)
- **JavaScript** (`clients/javascript/tests/test_client.js`)
- **Go** (`clients/go/test_client.go`)
- **Java** (`clients/java/src/main/java/moustique/TestClient.java`)
- **Perl** (`clients/perl/test_client.pl`)

## Prerequisites

Before running the tests, make sure you have:

1. **Moustique server running** with both HTTP and MQTT ports enabled
2. **User credentials** configured (or public access enabled)
3. Required language runtimes installed:
   - Python 3.7+
   - Node.js 18+
   - Go 1.22+
   - Java 17+ with Maven
   - Perl 5.10+

## Running All Tests

The easiest way to test all clients is using the master test script:

```bash
./tests/test_all_clients.sh [server_ip] [http_port] [username] [password] [mqtt_port]
```

**Examples:**

```bash
# Using defaults (localhost:33334, demo/demo123, MQTT port 1883)
./tests/test_all_clients.sh

# Custom server
./tests/test_all_clients.sh 192.168.1.100 33334 alice secret123 1883
```

## Running Individual Client Tests

### Python
```bash
cd clients/python
python3 tests/test_client.py localhost 33334 demo demo123 1883
```

### JavaScript
```bash
cd clients/javascript
npm install  # First time only
node tests/test_client.js localhost 33334 demo demo123 1883
```

### Go
```bash
cd clients/go
go mod tidy  # First time only
go run test_client.go localhost 33334 demo demo123 1883
```

### Java
```bash
cd clients/java
mvn clean package  # First time only
mvn exec:java -Dexec.mainClass=moustique.TestClient -Dexec.args="localhost 33334 demo demo123 1883"
```

### Perl
```bash
cd clients/perl
perl test_client.pl localhost 33334 demo demo123 1883
```

## What Each Test Does

Each test performs the following operations:

### HTTP Test
1. Publishes a message
2. Stores a key-value pair
3. Subscribes to a topic
4. Polls for messages (10 seconds)
5. Verifies callback triggers

### MQTT Test
1. Connects to MQTT broker with authentication
2. Publishes a message
3. Stores a key-value pair
4. Subscribes to a topic
5. Waits for pushed messages (10 seconds)
6. Verifies callback triggers
7. Disconnects cleanly

## Expected Output

Successful test output looks like:

```
============================================================
Testing with HTTP protocol
============================================================

=== Moustique Python Client – Multi-tenant Test ===
Protocol: HTTP
Server: localhost:33334
Username: demo

1. Publishing message...
2. Setting value...
3. Subscribing to /test/topic...
   Sending message to trigger callback...
   Polling for HTTP messages (10 seconds)...
[10:30:15] MESSAGE → '/test/topic': This message should appear in callback! (HTTP) (from TestRunner-http)

=== HTTP test complete! ===

============================================================
TEST SUMMARY
============================================================
HTTP test: ✓ PASSED
MQTT test: ✓ PASSED
============================================================
```

## Troubleshooting

### "MQTT requested but paho-mqtt not installed"
Install the MQTT library for your language:
- Python: `pip install paho-mqtt`
- JavaScript: `npm install mqtt`
- Go: `go get github.com/eclipse/paho.mqtt.golang`
- Java: Dependencies in pom.xml
- Perl: MQTT not currently supported (library limitations). The Perl client uses HTTP polling only and gracefully falls back when MQTT is requested.

### "Authentication failed"
- Verify username/password are correct
- Check server configuration (`allow_public` setting)
- Review server logs

### "Connection refused"
- Ensure Moustique server is running
- Verify HTTP and MQTT ports are correct
- Check firewall settings

### Tests timeout
- Increase wait time in test code
- Check server is processing messages
- Verify network connectivity

## Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed

## CI/CD Integration

Use the test script in your CI pipeline:

```yaml
# GitHub Actions example
- name: Test Moustique Clients
  run: |
    ./moustique &
    sleep 2
    ./tests/test_all_clients.sh localhost 33334 testuser testpass 1883
```
