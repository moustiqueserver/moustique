# 🦟 Moustique

**A lightweight, high-performance Pub/Sub Message Broker — speaks both MQTT and HTTP.**

[![License](https://img.shields.io/github/license/moustiqueserver/moustique)](https://opensource.org/licenses/gpl3-0)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![MQTT](https://img.shields.io/badge/MQTT-5.0-purple?logo=mqtt)](https://mqtt.org/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

Moustique is a simple, fast, and lightweight pub/sub message broker with **dual protocol support**: native MQTT for real-time push messaging and HTTP for polling-based clients. Choose the protocol that fits your use case — or use both simultaneously!

**Moustique** offers:

- 🔄 **Dual Protocol Support** - Native MQTT for push + HTTP for polling — use what fits your needs
- 🎯 **Simple integration** - Clients for Go, Python, JavaScript, Java, Perl with auto-detect MQTT/HTTP
- 🚀 **High performance** - Written in Go, handles thousands of concurrent connections
- 📡 **Real-time push** – MQTT delivers messages instantly without polling
- 📨 **HTTP polling fallback** – Works everywhere, even through restrictive firewalls
- 🔑 **Key/Value storage** – Store and retrieve persistent values
- 💾 **Persistent storage** - Messages survive restarts with SQLite backend
- 🎨 **Built-in web UI** - Monitor and manage your broker from your browser
- 🔍 **Powerful wildcards** - MQTT-style topic matching with `+` and `#`
- 🔐 **Multi-tenant support** - Optional per-user authentication and isolated brokers
- 🛠️ **Command-line tools** - Standalone CLI for quick interactions
- ⚡ **Compatible** - Works with standard MQTT clients like `mosquitto_pub`/`mosquitto_sub`

## 🎬 See It In Action

Check out our [interactive demos](demos/) to see Moustique in action, or try the Quick Start below:

## 🚀 Quick Start

### Installation

```bash
# Download binary (Linux/macOS/Windows)
curl -L https://github.com/moustiqueserver/moustique/releases/latest/download/moustique-linux-amd64 -o moustique
chmod +x moustique

# Or build from source
git clone https://github.com/moustiqueserver/moustique.git
cd moustique
make all
```

### Run the server

```bash
# Start with defaults (HTTP port 33334, MQTT port 1883)
./moustique

# Or with custom config
./moustique -config myconfig.yaml

# Generate default config
./moustique -generate-config
```

**The server now listens on TWO ports:**
- **HTTP**: `http://localhost:33334` - Web UI, REST API, polling clients
- **MQTT**: `tcp://localhost:1883` - Standard MQTT protocol for push messaging

**Open web UI:**
```bash
# Open in browser
http://localhost:33334/
```

### Try it with MQTT

Moustique works with standard MQTT tools:

```bash
# Subscribe with mosquitto
mosquitto_sub -h localhost -p 1883 -t "test/topic" -u demo -P demo123

# Publish with mosquitto (in another terminal)
mosquitto_pub -h localhost -p 1883 -t "test/topic" -m "Hello MQTT!" -u demo -P demo123

# Also works with HTTP
curl -X POST http://localhost:33334/POST \
  -d "topic=$(echo -n 'test/topic' | base64 | tr 'A-Za-z' 'N-ZA-Mn-za-m')" \
  -d "message=$(echo -n 'Hello HTTP!' | base64 | tr 'A-Za-z' 'N-ZA-Mn-za-m')"
```

### Command-line Client

Moustique includes a powerful CLI tool for interacting with the broker:

```bash
# Publish a message
./moustique-cli -a pub -t /test/topic -m "Hello World"

# Subscribe to a topic
./moustique-cli -a sub -t /test/topic

# Store a value
./moustique-cli -a put -t /config/setting -m "value123"

# With authentication
./moustique-cli -a pub -u alice -pwd secret123 -t /private/topic -m "Secure message"

# Connect to remote server
./moustique-cli -h moustique.example.com -p 33334 -a pub -t /remote/topic -m "Hi"
```

See [CLI documentation](cmd/moustique-cli/README.md) for more details.

## 📚 Client Libraries

Moustique has official clients for the most popular programming languages:

### Python

**Installation:**
```bash
pip install moustique-client
```

**Usage:**
```python
from moustique import Moustique
import time

# Create client with MQTT (real-time push)
client = Moustique(
    ip="127.0.0.1",
    port="33334",
    client_name="MyApp",
    username="demo",
    password="demo123",
    use_mqtt=True,        # Enable MQTT for instant message delivery
    mqtt_port=1883
)

# Subscribe to messages
def on_message(topic, message, from_name):
    print(f"Message on {topic}: {message} from {from_name}")

client.subscribe("/test/topic", on_message)

# Publish message
client.publish("/test/topic", "Hello from Python!")

# Store value
client.putval("/config/setting", "value")

# Get value
value = client.get_val("/config/setting")

# With MQTT: messages arrive instantly via callbacks
# With HTTP: need to poll for messages
while True:
    client.tick()  # MQTT: no-op, HTTP: polls for messages
    time.sleep(1)

# Clean disconnect
client.disconnect()
```

**HTTP-only mode (no MQTT):**
```python
# Omit use_mqtt parameter or set it to False
client = Moustique(ip="127.0.0.1", port="33334", client_name="MyApp")
# Client will use HTTP polling automatically
```

**Helper functions:**
```python
from moustique import getversion, getstats, getclients

# Get server info
version = getversion("127.0.0.1", "33335", "password")
stats = getstats("127.0.0.1", "33335", "password")
clients = getclients("127.0.0.1", "33335", "password")
```

### JavaScript/Node.js

**Installation:**
```bash
npm install moustique-client
```

**Usage:**
```javascript
import { Moustique } from 'moustique-client';

// Create client with MQTT (real-time push)
const client = new Moustique({
    ip: '127.0.0.1',
    port: '33334',
    clientName: 'MyApp',
    username: 'demo',
    password: 'demo123',
    useMqtt: true,      // Enable MQTT for instant delivery
    mqttPort: 1883
});

// Subscribe to messages
client.subscribe('/test/topic', (topic, message, from) => {
    console.log(`Message on ${topic}: ${message} from ${from}`);
});

// Publish message
await client.publish('/test/topic', 'Hello from JavaScript!');

// Store value
await client.putval('/config/setting', 'value');

// Get value
const value = await client.getval('/config/setting');

// With MQTT: messages arrive automatically via callbacks
// With HTTP: need to poll for messages
if (!client.useMqtt) {
    setInterval(() => client.pickup(), 1000);
}

// Clean disconnect
client.disconnect();
```

### Java

**Installation (Maven):**
```xml
<dependency>
    <groupId>com.moustique</groupId>
    <artifactId>moustique-client</artifactId>
    <version>1.0.0</version>
</dependency>
```

**Installation (Gradle):**
```gradle
implementation 'com.moustique:moustique-client:1.0.0'
```

**Usage:**
```java
import moustique.MoustiqueClient;

// Create client with MQTT (real-time push)
MoustiqueClient client = new MoustiqueClient(
    "127.0.0.1",
    "33334",
    "MyApp",
    "demo",      // username
    "demo123",   // password
    true,        // use MQTT
    1883         // MQTT port
);

// Subscribe to messages
client.subscribe("/test/topic", msg -> {
    System.out.println(msg.topic() + ": " + msg.message() + " from " + msg.from());
});

// Publish message
client.publish("/test/topic", "Hello from Java!").join();

// Store value
client.putval("/config/setting", "value").join();

// Get value
String value = client.getval("/config/setting").join();

// With MQTT: messages arrive via callbacks
// With HTTP: poll for messages
while (true) {
    client.pickup().join();  // MQTT: no-op, HTTP: polls
    TimeUnit.SECONDS.sleep(1);
}

// Clean disconnect
client.disconnect();
```

### Go

**Installation:**
```bash
go get github.com/moustiqueserver/moustique/clients/go/moustique
```

**Usage:**
```go
import "github.com/moustiqueserver/moustique/clients/go/moustique"

// Create client with MQTT (real-time push)
client := moustique.New(
    "127.0.0.1",
    "33334",
    "MyApp",
    "demo",      // username
    "demo123",   // password
    "true",      // use MQTT
    "1883"       // MQTT port
)

// Subscribe to messages
client.Subscribe("/test/topic", func(topic, message, from string) {
    fmt.Printf("%s: %s from %s\n", topic, message, from)
})

// Publish message
client.Publish("/test/topic", "Hello from Go!")

// Store value
client.PutVal("/config/setting", "value")

// Get value
value := client.GetVal("/config/setting")

// With MQTT: messages arrive via callbacks
// With HTTP: poll for messages
ticker := time.NewTicker(1 * time.Second)
for range ticker.C {
    client.Pickup()  // MQTT: no-op, HTTP: polls
}

// Clean disconnect
client.Disconnect()
```

**HTTP-only mode:**
```go
// Omit MQTT parameters for HTTP polling
client := moustique.New("127.0.0.1", "33334", "MyApp", "demo", "demo123")
```
### Perl

**Usage:**
```perl
use Moustique;

# Create client (HTTP polling)
my $mous = Moustique->new(
    ip => "localhost",
    port => 33334,
    name => "my-app",
    username => "demo",
    password => "demo123"
);

# Subscribe to messages
$mous->subscribe("/sensors/+/temperature", sub {
    my ($topic, $message, $from) = @_;
    print "Temperature: $message from $from\n";
});

# Publish message
$mous->publish("/sensors/bedroom/temperature", "23.1", "my-app");

# Poll for messages
while (1) {
    $mous->tick();
    sleep(1);
}
```

**Note:** MQTT not currently supported in Perl client due to library limitations. The Perl client uses HTTP polling and gracefully falls back when MQTT is requested. For production MQTT use, consider the Python, JavaScript, Go, or Java clients.

## 🎯 Key Features

### 1. Hybrid MQTT + HTTP Architecture

Moustique uniquely supports **both** protocols simultaneously:

```
┌─────────────────────────────────────────────────────────┐
│                  Moustique Broker                       │
│  ┌────────────────┐        ┌────────────────────┐      │
│  │  HTTP Server   │        │   MQTT Server      │      │
│  │  Port 33334    │        │   Port 1883        │      │
│  └────────┬───────┘        └──────┬─────────────┘      │
│           │                       │                     │
│           └───────┬───────────────┘                     │
│                   │                                     │
│           ┌───────▼────────┐                            │
│           │  Message Bus   │                            │
│           │  (Per-User)    │                            │
│           └────────────────┘                            │
└─────────────────────────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
   ┌────▼───┐   ┌───▼────┐   ┌──▼─────┐
   │ Python │   │ Node.js│   │mosquitto│
   │ (MQTT) │   │ (HTTP) │   │  (MQTT) │
   └────────┘   └────────┘   └────────┘
```

**Choose your protocol:**
- **MQTT** - Real-time push, low latency, efficient for IoT
- **HTTP** - Works everywhere, simple REST API, firewall-friendly
- **Both** - MQTT client publishes, HTTP client receives (or vice versa)

**Why hybrid?**
- **Flexibility**: Each client chooses the best protocol for their needs
- **Compatibility**: Works with standard MQTT tools AND REST clients
- **Reliability**: HTTP fallback when MQTT is blocked by firewalls
- **Migration**: Gradually migrate from HTTP to MQTT without breaking changes

### 2. Multi-Tenant Support

Moustique supports optional authentication with isolated brokers per user:

```bash
# Enable public access (no auth required) in config.yaml
server:
  allow_public: true

# Or require authentication
server:
  allow_public: false
```

**Using authentication in clients:**

```python
# Python with authentication
client = Moustique(ip="127.0.0.1", port="33334",
                   client_name="MyApp",
                   username="alice",
                   password="secret123")
```

```javascript
// JavaScript with authentication
const client = new Moustique({
    ip: '127.0.0.1',
    port: '33334',
    clientName: 'MyApp',
    username: 'alice',
    password: 'secret123'
});
```

```bash
# CLI with authentication
./moustique-cli -a pub -u alice -pwd secret123 -t /topic -m "message"
```

Each authenticated user gets their own isolated broker, preventing cross-tenant data access.

### 2. Wildcard Subscriptions

Subscribe to multiple topics with MQTT-style wildcards:

```bash
/home/sensors/+/temperature     # Matches any room
/home/sensors/#                 # Matches everything under sensors
/home/+/+/humidity              # Multi-level wildcards
```

### 3. Persistent Storage

Messages are stored in SQLite and survive server restarts:

```bash
# Get stored value
curl http://localhost:33335/GETVAL?topic=ENCODED_TOPIC

# Search by regex
curl http://localhost:33335/GETVALSBYREGEX?topic=ENCODED_REGEX
```

### 4. Built-in Monitoring

Beautiful web UI at `http://localhost:33335/` shows:
- Real-time statistics (per-second rates)
- Active clients and publishers
- All topics and subscriptions
- Message throughput
- Per-user broker statistics (multi-tenant mode)
- Server logs and user-specific logs

### 5. Automatic Reconnection

Clients automatically resubscribe after server restarts—no manual intervention needed.

### 6. Lightweight & Fast

- **Small footprint**: ~10MB binary, ~20MB RAM usage
- **High throughput**: Handles 10,000+ messages/second
- **Low latency**: Sub-millisecond message delivery
- **Concurrent**: Supports 1000+ simultaneous connections

## 📖 Documentation

### Configuration

Create `config.yaml`:

```yaml
server:
  port: 33334              # HTTP port for web UI and API
  mqtt_port: 1883          # MQTT port for push messaging (set to 0 to disable)
  host: "0.0.0.0"
  timeout: 30s
  allow_public: false      # Require authentication
  max_request_size: 10485760  # 10MB

database:
  path: "./data"           # SQLite database directory

logging:
  level: "info"
  file: ""                 # Empty = console only, or specify path

security:
  allowed_peers:
    - "192.168.0.0/16"
    - "10.0.0.0/8"
    - "172.16.0.0/12"
  blocked_peers: []
  max_topic_length: 256
  max_message_size: 1048576  # 1MB
  default_rate_limit: 1000   # Requests per minute (0 = unlimited)
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/SUBSCRIBE` | POST | Subscribe to a topic |
| `/POST` | POST | Publish a message |
| `/PICKUP` | POST | Get pending messages |
| `/GETVAL` | POST | Get stored value |
| `/GETVALSBYREGEX` | POST | Search values by pattern |
| `/STATUS` | POST | Get broker status (auth required) |
| `/STATS` | POST | Get statistics (auth required) |
| `/CLIENTS` | POST | List active clients (auth required) |
| `/TOPICS` | POST | List all topics (auth required) |

### Encoding

Moustique uses ROT13+Base64 encoding for a lightweight security layer:

```bash
# Encode
echo -n "my-topic" | base64 | tr 'A-Za-z' 'N-ZA-Mn-za-m'

# Decode  
echo "encoded" | tr 'A-Za-z' 'N-ZA-Mn-za-m' | base64 -d
```

Client libraries handle this automatically.

## 🐳 Docker

```bash
# Run with Docker
docker run -p 33335:33335 -v $(pwd)/data:/data moustique/moustique

# Docker Compose
docker-compose up -d
```

## 🛡️ Security & Monitoring

### Logging

Moustique creates three separate log files in the configured logging directory:

- **`moustique.log`** - Main server log (debug, info, warnings, errors)
- **`moustique_access.log`** - All HTTP requests with timing information
- **`moustique_error.log`** - Security events and errors only

**Access log format:**
```
2025-12-28 00:58:12 | ::1 | POST | /POST | username | 200 | 1.23ms
```

**Error log format:**
```
2025-12-28 00:58:33 | ::1 | invalid_endpoint | Invalid API endpoint: SCAN
2025-12-28 00:58:45 | 192.168.1.100 | invalid_credentials | Failed login: baduser
2025-12-28 00:59:01 | 10.0.0.50 | rate_limit_exceeded | User alice exceeded 1000 req/min
```

**Log rotation:**
- Each log file is automatically rotated at 3MB
- Keeps 2 old files per log type (e.g., `moustique_error.log.1`, `moustique_error.log.2`)
- Maximum total disk usage: ~27MB (3 logs × 3 files × 3MB)

### Fail2ban Integration

Moustique includes built-in fail2ban integration with configurable strictness levels:

**Configuration:**
```yaml
security:
  fail2ban_jail: "moustique"        # Jail name (empty = disabled)
  fail2ban_level: "normal"          # Strictness level
```

**Fail2ban levels:**

| Level | Bans on | Use case |
|-------|---------|----------|
| `minimal` | Endpoint scanning only | Public-facing servers with many users |
| `relaxed` | Scanning + invalid credentials | Semi-public servers |
| `normal` | + validation errors | **Default** - Balanced security |
| `strict` | + oversized requests + rate limits | High-security environments |

**Tracked violations:**
- `invalid_endpoint` - API endpoint scanning attempts
- `invalid_credentials` - Failed login attempts
- `validation_error` - Malformed data (oversized topics/messages)
- `oversized_request` - Request body exceeds limits
- `rate_limit_exceeded` - Too many requests per minute

**Setup fail2ban jail:**

Create `/etc/fail2ban/jail.d/moustique.conf`:
```ini
[moustique]
enabled = true
port = 33334
logpath = /var/log/moustique/moustique_error.log
maxretry = 3
findtime = 600
bantime = 3600
```

Create `/etc/fail2ban/filter.d/moustique.conf`:
```ini
[Definition]
failregex = ^.* \| <HOST> \| (invalid_endpoint|invalid_credentials|validation_error) \|
ignoreregex =
```

Then reload fail2ban:
```bash
sudo systemctl reload fail2ban
```

### Rate Limiting

Configure per-user rate limits via web UI (`/superadmin`) or config:

```yaml
security:
  default_rate_limit: 1000  # Requests per minute (0 = unlimited)
```

Rate limits can be set individually for each user through the admin interface.

## 🔧 Production Deployment

### systemd Service

```ini
[Unit]
Description=Moustique Message Broker
After=network.target

[Service]
Type=simple
User=moustique
ExecStart=/usr/local/bin/moustique -config /etc/moustique/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Behind Nginx

```nginx
location /moustique/ {
    proxy_pass http://localhost:33335/;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
}
```

## 🤝 Contributing

Contributions are welcome! Here's how to help:

1. 🍴 Fork the repository
2. 🌱 Create a feature branch (`git checkout -b feature/amazing`)
3. 💾 Commit your changes (`git commit -m 'Add amazing feature'`)
4. 📤 Push to branch (`git push origin feature/amazing`)
5. 🎉 Open a Pull Request

### Development Setup

```bash
git clone https://github.com/moustiqueserver/moustique.git
cd moustique

# Build everything
make all

# Or build specific components
make server      # Build server only
make cli         # Build CLI only

# Run with debug mode
./moustique -debug
```

### Running Tests

```bash
# Run Go unit tests
make test

# Run client integration tests (requires running server)
# First, start the server in another terminal:
./moustique

# Then run client tests:
make test-clients

# Or manually:
./tests/test_all_clients.sh
```

See [tests/README.md](tests/README.md) for detailed testing documentation.

### Building for Multiple Platforms

```bash
# Build all platforms
make dist-all

# Build specific platforms
make server-linux    # Linux AMD64 and ARM64
make server-darwin   # macOS Intel and Apple Silicon
make server-windows  # Windows AMD64
make cli-linux       # Linux (AMD64, ARM64, ARM)
make cli-darwin      # macOS (Intel, Apple Silicon)
make cli-windows     # Windows AMD64

# Install locally
sudo make install
```

See the [Makefile](Makefile) for all available targets.

## 📊 Performance

Benchmarks on a modest server (4 CPU cores, 8GB RAM):

| Metric | Value |
|--------|-------|
| Messages/sec | 12,000+ |
| Concurrent clients | 1,000+ |
| Latency (p50) | <1ms |
| Latency (p99) | <5ms |
| Memory usage | ~50MB @ 1000 clients |

## 🗺️ Roadmap

- [x] Core pub/sub functionality
- [x] Wildcard subscriptions
- [x] Persistent storage
- [x] Web UI
- [x] Multi-tenant support with authentication
- [x] Command-line client (moustique-cli)
- [x] JavaScript/TypeScript client
- [x] Python client
- [x] Go client
- [x] Java client
- [x] Perl client
- [x] **MQTT protocol support** 🎉
- [x] Hybrid MQTT/HTTP clients with automatic fallback
- [x] Standard MQTT compatibility (mosquitto_pub/sub)
- [x] Rate limiting per user
- [x] IP-based access control (CIDR support)
- [ ] TLS/HTTPS support (HTTP + MQTTS)
- [ ] MQTT QoS 1 & 2 (currently QoS 0 only)
- [ ] Retained messages
- [ ] Last Will and Testament (LWT)
- [ ] Message retention policies
- [ ] Clustering support
- [ ] WebSocket support
- [ ] Authentication plugins (JWT, OAuth2)
- [ ] Prometheus metrics endpoint
- [ ] NOSTR relay support

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with love using Go

## 💬 Community

- 📣 [Discussions](https://github.com/yourusername/moustique/discussions)
- 🐛 [Issue Tracker](https://github.com/yourusername/moustique/issues)
- 💡 [Feature Requests](https://github.com/yourusername/moustique/issues/new?template=feature_request.md)

---

⭐ **Star us on GitHub** if Moustique makes your life easier!
