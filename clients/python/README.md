# 🦟 Moustique

**A lightweight, high-performance message broker that speaks HTTP.**

[![License](https://img.shields.io/github/license/moustiqueserver/moustique)](GNU GPLv3)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

Moustique is a simple, fast, and lightweight pub/sub message broker that uses plain HTTP(S) for communication.

**Moustique** offers:

- 🎯 **Simple integration** - Easy to use clients available for Go, Python, JavaScript, and even Perl
- 🚀 **High performance** - Written in Go, handles thousands of concurrent connections
- 📡 **Pub/Sub communication** – subscribe to topics and receive messages in real-time
- 🔑 **Key/Value storage** – store and retrieve values
- 💾 **Persistent storage** - Messages survive restarts with SQLite backend
- 🎨 **Built-in web UI** - Monitor and manage your broker from your browser
- 🔍 **Powerful wildcards** - MQTT-style topic matching with `+` and `#`

## 🚀 Quick Start

### Installation

```bash
# Download binary (Linux/macOS/Windows)
curl -L https://github.com/moustiqueserver/moustique/releases/latest/download/moustique-linux-amd64 -o moustique
chmod +x moustique

# Or build from source
git clone https://github.com/moustiqueserver/moustique.git
cd moustique
go build
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

### 1. Wildcard Subscriptions

Subscribe to multiple topics with MQTT-style wildcards:

```bash
/home/sensors/+/temperature     # Matches any room
/home/sensors/#                 # Matches everything under sensors
/home/+/+/humidity              # Multi-level wildcards
```

### 2. Persistent Storage

Messages are stored in SQLite and survive server restarts:

```bash
# Get stored value
curl http://localhost:33335/GETVAL?topic=ENCODED_TOPIC

# Search by regex
curl http://localhost:33335/GETVALSBYREGEX?topic=ENCODED_REGEX
```

### 3. Built-in Monitoring

Beautiful web UI at `http://localhost:33335/` shows:
- Real-time statistics
- Active clients and publishers
- All topics and subscriptions
- Message throughput

### 4. Automatic Reconnection

Clients automatically resubscribe after server restarts—no manual intervention needed.

### 5. Lightweight & Fast

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
  file: "./logs/moustique.log"

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
go build
./moustique -debug
```

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
- [x] JavaScript/TypeScript client
- [x] Python client
- [x] Go client
- [x] Java client
- [ ] TLS/HTTPS support
- [ ] Authentication plugins
- [ ] Message retention policies

## 📜 License

GNU GPLv3 License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with love using Go

## 💬 Community

- 📣 [Discussions](https://github.com/moustiqueserver/moustique/discussions)
- 🐛 [Issue Tracker](https://github.com/moustiqueserver/moustique/issues)
- 💡 [Feature Requests](https://github.com/moustiqueserver/moustique/issues/new?template=feature_request.md)

---

⭐ **Star us on GitHub** if Moustique makes your life easier!
