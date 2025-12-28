# 🦟 Moustique

**A lightweight, high-performance message broker that speaks HTTP.**

[![License](https://img.shields.io/github/license/moustiqueserver/moustique)](https://opensource.org/licenses/gpl3-0)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

Moustique is a simple, fast, and lightweight pub/sub message broker that uses plain HTTP(S) for communication.

**Moustique** offers:

- 🎯 **Simple integration** - Easy to use clients available for Go, Python, JavaScript, Java, Perl, and CLI
- 🚀 **High performance** - Written in Go, handles thousands of concurrent connections
- 📡 **Pub/Sub communication** – subscribe to topics and receive messages in real-time
- 🔑 **Key/Value storage** – store and retrieve values
- 💾 **Persistent storage** - Messages survive restarts with SQLite backend
- 🎨 **Built-in web UI** - Monitor and manage your broker from your browser
- 🔍 **Powerful wildcards** - MQTT-style topic matching with `+` and `#`
- 🔐 **Multi-tenant support** - Optional per-user authentication and isolated brokers
- 🛠️ **Command-line tools** - Standalone CLI for quick interactions

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
# Start with defaults (port 33335)
./moustique

# Or with custom config
./moustique -config myconfig.yaml

# Generate default config
./moustique -generate-config
```

**Open web UI:**
```bash
# Open in browser
http://localhost:33335/
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

# Create client
client = Moustique(ip="127.0.0.1", port="33335", client_name="MyApp")

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

# Poll for new messages (run in loop)
while True:
    client.tick()
    time.sleep(1)
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

// Create client
const client = new Moustique({
    ip: '127.0.0.1',
    port: '33335',
    clientName: 'MyApp'
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

// Poll for new messages
setInterval(() => client.pickup(), 1000);
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

// Create client
MoustiqueClient client = new MoustiqueClient("127.0.0.1", "33335", "MyApp");

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

// Poll for new messages
while (true) {
    client.pickup().join();
    TimeUnit.SECONDS.sleep(1);
}
```

### Go

**Installation:**
```bash
go get github.com/moustiqueserver/moustique/clients/go/moustique
```

**Usage:**
```go
import "github.com/moustiqueserver/moustique/clients/go/moustique"

// Create client
client := moustique.New("127.0.0.1", "33335", "MyApp")

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

// Poll for new messages
ticker := time.NewTicker(1 * time.Second)
for range ticker.C {
    client.Pickup()
}
```
### Perl
**Usage:**
```perl
use Moustique;

my $mous = Moustique->new(ip => "localhost", port => 33335, name => "my-app");
$mous->subscribe("/sensors/+/temperature", sub {
    my ($topic, $message) = @_;
    print "Temperature: $message\n";
});
$mous->publish("/sensors/bedroom/temperature", "23.1");
```

## 🎯 Key Features

### 1. Multi-Tenant Support

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
  port: 33335
  host: "0.0.0.0"
  timeout: 5s
  max_connections: 1000

database:
  path: "./data/moustique.db"

security:
  allowed_ips:
    - "192.168.0.0/16"
    - "10.0.0.0/8"
  tailscale_enabled: true
  password_file: "./data/.moustique_pwd"

logging:
  level: "info"
  directory: "/var/log/moustique"  # Directory for all log files

security:
  allowed_peers:
    - "192.168.0.0/16"
    - "10.0.0.0/8"
    - "172.16.0.0/12"
  max_topic_length: 256
  max_message_size: 1048576  # 1MB
  default_rate_limit: 1000   # Requests per minute (0 = unlimited)
  fail2ban_jail: "moustique"
  fail2ban_level: "normal"   # minimal, relaxed, normal, strict

performance:
  message_queue_timeout: 5m
  poster_stats_timeout: 1h
  maintenance_interval: 30s
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
- [x] Access logging with rotation
- [x] Fail2ban integration with configurable levels
- [x] Rate limiting per user
- [x] IP-based access control (CIDR support)
- [ ] TLS/HTTPS support
- [ ] Authentication plugins
- [ ] Message retention policies
- [ ] Clustering support
- [ ] WebSocket support
- [ ] MQTT protocol support
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
