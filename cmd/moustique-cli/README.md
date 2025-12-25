# Moustique CLI

Command-line client for the Moustique message broker.

## Installation

Build the CLI tool:

```bash
go build -o moustique-cli ./cmd/moustique-cli
```

Or install it to your $GOPATH/bin:

```bash
go install ./cmd/moustique-cli
```

## Usage

```
moustique-cli -a <action> [options]
```

### Actions

- `pub`, `publish` - Publish a message to a topic
- `put`, `putval` - Store a key-value pair
- `sub`, `subscribe` - Subscribe to a topic and listen for messages
- `version` - Show version information

### Options

- `-a string` - Action to perform (required)
- `-h string` - Moustique server host (default: localhost)
- `-p string` - Moustique server port (default: 33334)
- `-t string` - Topic
- `-m string` - Message
- `-n string` - Client name (auto-generated if not provided)
- `-u string` - Username for authentication (optional)
- `-pwd string` - Password for authentication (optional)
- `-v` - Verbose output
- `-help` - Show help message

## Examples

### Publish to public broker

```bash
moustique-cli -a pub -t /test/topic -m "Hello World"
```

### Publish with authentication

```bash
moustique-cli -a pub -u alice -pwd secret123 -t /test/topic -m "Hello"
```

### Subscribe to topic

```bash
moustique-cli -a sub -t /test/topic
```

Output format: `timestamp | topic | from | message`

### Subscribe with authentication

```bash
moustique-cli -a sub -u alice -pwd secret123 -t /private/topic
```

### Put a value

```bash
moustique-cli -a put -t /config/setting -m "value123"
```

### Connect to remote server

```bash
moustique-cli -h moustique.host -p 33334 -a pub -t /remote/topic -m "Hi"
```

## Cross-compilation

Build for different platforms:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o moustique-cli-linux-amd64 ./cmd/moustique-cli

# macOS ARM64 (M1/M2)
GOOS=darwin GOARCH=arm64 go build -o moustique-cli-darwin-arm64 ./cmd/moustique-cli

# macOS AMD64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o moustique-cli-darwin-amd64 ./cmd/moustique-cli

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o moustique-cli-windows-amd64.exe ./cmd/moustique-cli

# Linux ARM64 (Raspberry Pi, ARM servers)
GOOS=linux GOARCH=arm64 go build -o moustique-cli-linux-arm64 ./cmd/moustique-cli

# Linux ARM (32-bit Raspberry Pi)
GOOS=linux GOARCH=arm go build -o moustique-cli-linux-arm ./cmd/moustique-cli
```

## Authentication

The CLI supports optional authentication. If you don't provide `-u` and `-pwd`, it will connect to the public broker (if enabled on the server).

If authentication is required, provide both username and password:

```bash
moustique-cli -a pub -u myuser -pwd mypassword -t /topic -m "message"
```

## Notes

- The CLI creates a unique client name by appending a UUID to your hostname (or custom name)
- Messages are encoded using ROT13 + Base64 for obfuscation (same as other Moustique clients)
- The subscribe action runs continuously until you press Ctrl+C
- One-second polling interval for subscriptions
