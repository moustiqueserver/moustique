# Moustique Go Client

[![Go Reference](https://pkg.go.dev/badge/github.com/moustiqueserver/moustique/clients/go/moustique.svg)](https://pkg.go.dev/github.com/moustiqueserver/moustique/clients/go/moustique)
[![Go Report Card](https://goreportcard.com/badge/github.com/moustiqueserver/moustique/clients/go)](https://goreportcard.com/report/github.com/moustiqueserver/moustique/clients/go)
[![License](https://img.shields.io/github/license/moustiqueserver/moustique)](https://github.com/moustiqueserver/moustique/blob/main/LICENSE.md)

A lightweight, zero-dependency HTTP-based **publish/subscribe client** for the [Moustique message broker](https://github.com/moustiqueserver/moustique).

## Features

- Full **publish/subscribe** functionality
- Persistent key-value storage via `PutVal`/`GetVal`
- Automatic **resubscribe** on reconnect
- Zero external dependencies (except uuid)
- Thread-safe operations
- Simple, idiomatic Go API

## Installation

```bash
go get github.com/moustiqueserver/moustique/clients/go/moustique
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/moustiqueserver/moustique/clients/go/moustique"
)

func main() {
    // Create client
    client := moustique.New("127.0.0.1", "33334", "my-app", "", "")

    // Subscribe to messages
    client.Subscribe("/test/topic", func(topic, message, from string) {
        fmt.Printf("Received on %s: %s (from %s)\n", topic, message, from)
    })

    // Publish message
    err := client.Publish("/test/topic", "Hello from Go!", "my-app")
    if err != nil {
        panic(err)
    }

    // Store value
    err = client.PutVal("/config/setting", "value", "my-app")
    if err != nil {
        panic(err)
    }

    // Get value
    value, err := client.GetVal("/config/setting")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Got value: %s\n", value)

    // Poll for new messages
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        client.Pickup()
    }
}
```

## API Reference

### Creating a Client

```go
client := moustique.New(ip, port, clientName, username, password)
```

- `ip`: Server IP address (e.g., "127.0.0.1")
- `port`: Server port (e.g., "33334")
- `clientName`: Your application name
- `username`: Optional username for authentication (empty string for public broker)
- `password`: Optional password for authentication (empty string for public broker)

### Publishing Messages

```go
err := client.Publish(topic, message, from)
```

Publishes a message to a topic.

### Subscribing to Topics

```go
client.Subscribe(topic, callback)
```

Subscribe to a topic with a callback function:

```go
client.Subscribe("/sensors/temperature", func(topic, message, from string) {
    fmt.Printf("Temperature: %s\n", message)
})
```

Supports MQTT-style wildcards:
- `+` - Single-level wildcard (e.g., `/sensors/+/temperature`)
- `#` - Multi-level wildcard (e.g., `/sensors/#`)

### Key-Value Storage

```go
// Store value
err := client.PutVal(key, value, from)

// Retrieve value
value, err := client.GetVal(key)
```

### Picking Up Messages

```go
client.Pickup()
```

Poll for new messages. Call this regularly (e.g., in a ticker) to receive subscribed messages.

## Authentication

For authenticated brokers, provide username and password:

```go
client := moustique.New("127.0.0.1", "33334", "my-app", "username", "password")
```

For public brokers, use empty strings:

```go
client := moustique.New("127.0.0.1", "33334", "my-app", "", "")
```

## Error Handling

All network operations return errors that should be checked:

```go
if err := client.Publish("/topic", "message", "app"); err != nil {
    log.Printf("Failed to publish: %v", err)
}
```

## Examples

See [example_test.go](moustique/example_test.go) for more examples.

## Thread Safety

The client is thread-safe and can be used from multiple goroutines concurrently.

## License

GNU GPLv3 License - see the [LICENSE](../../LICENSE.md) file for details.

## Links

- [Moustique Server](https://github.com/moustiqueserver/moustique)
- [Go Package Documentation](https://pkg.go.dev/github.com/moustiqueserver/moustique/clients/go/moustique)
- [Report Issues](https://github.com/moustiqueserver/moustique/issues)
