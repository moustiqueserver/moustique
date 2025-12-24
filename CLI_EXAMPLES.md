# Moustique CLI - Usage Examples

Quick reference guide for the `moustique-cli` command-line tool.

## Installation

```bash
# Build from source
make cli

# Or for specific platform
make cli-linux
make cli-darwin
make cli-windows
```

## Basic Usage

### Publish Messages

```bash
# Simple publish to public broker
./moustique-cli -a pub -t /sensors/temperature -m "23.5"

# Publish with authentication
./moustique-cli -a pub -u alice -pwd secret123 -t /private/data -m "sensitive info"

# Publish to remote server
./moustique-cli -h mqtt.example.com -p 33334 -a pub -t /remote/sensor -m "data"
```

### Subscribe to Topics

```bash
# Subscribe to a topic
./moustique-cli -a sub -t /sensors/temperature

# Subscribe with authentication
./moustique-cli -a sub -u alice -pwd secret123 -t /private/updates

# Subscribe with verbose output (shows client name)
./moustique-cli -a sub -v -t /logs/#
```

Output format:
```
2025-12-24 12:30:45 | /sensors/temperature | device-001 | 23.5
2025-12-24 12:30:46 | /sensors/humidity | device-002 | 65
```

### Store and Retrieve Values

```bash
# Store a value
./moustique-cli -a put -t /config/api_key -m "sk_live_abc123"

# Store with authentication
./moustique-cli -a put -u bob -pwd pass456 -t /user/settings -m '{"theme":"dark"}'
```

## Real-World Examples

### IoT Sensor Network

```bash
# Temperature sensor publishing
./moustique-cli -a pub -t /home/bedroom/temperature -m "21.5"

# Subscribe to all home sensors
./moustique-cli -a sub -t /home/#

# Subscribe to specific sensor type across all rooms
./moustique-cli -a sub -t /home/+/temperature
```

### Log Aggregation

```bash
# Application publishing logs
./moustique-cli -a pub -t /logs/app1/error -m "Database connection failed"

# Subscribe to all errors
./moustique-cli -a sub -t /logs/+/error

# Subscribe to everything from app1
./moustique-cli -a sub -t /logs/app1/#
```

### Configuration Management

```bash
# Store configuration
./moustique-cli -a put -t /config/db_host -m "postgres.example.com"
./moustique-cli -a put -t /config/db_port -m "5432"
./moustique-cli -a put -t /config/max_connections -m "100"

# Applications can subscribe to config changes
./moustique-cli -a sub -t /config/#
```

### Inter-Service Communication

```bash
# Service A publishes an event
./moustique-cli -u serviceA -pwd secretA -a pub -t /events/order_created -m '{"order_id":"12345"}'

# Service B subscribes to events
./moustique-cli -u serviceB -pwd secretB -a sub -t /events/#
```

## Integration with Shell Scripts

### Bash Script Example

```bash
#!/bin/bash
# Monitor temperature and send alerts

BROKER_HOST="localhost"
BROKER_PORT="33334"
THRESHOLD=25

# Read from sensor (example)
TEMP=$(cat /sys/class/thermal/thermal_zone0/temp)
TEMP_C=$((TEMP/1000))

# Publish temperature
./moustique-cli -h $BROKER_HOST -p $BROKER_PORT \
    -a pub -t /monitoring/cpu/temperature -m "$TEMP_C"

# Alert if too high
if [ $TEMP_C -gt $THRESHOLD ]; then
    ./moustique-cli -h $BROKER_HOST -p $BROKER_PORT \
        -a pub -t /alerts/high_temperature -m "CPU temp is ${TEMP_C}°C"
fi
```

### Cron Job Example

```bash
# Add to crontab: Run every minute
* * * * * /usr/local/bin/moustique-cli -a pub -t /heartbeat/server1 -m "$(date +%s)"

# Monitor uptime every 5 minutes
*/5 * * * * /usr/local/bin/moustique-cli -a put -t /stats/uptime -m "$(uptime)"
```

### Python Integration

```python
import subprocess
import json

def publish_metrics(metrics):
    """Publish metrics using CLI"""
    result = subprocess.run([
        './moustique-cli',
        '-a', 'pub',
        '-u', 'metrics-service',
        '-pwd', 'secret',
        '-t', '/metrics/app',
        '-m', json.dumps(metrics)
    ], capture_output=True, text=True)

    if result.returncode != 0:
        print(f"Error: {result.stderr}")
    else:
        print(f"Published: {result.stdout}")

# Usage
publish_metrics({
    'cpu': 45.2,
    'memory': 62.8,
    'disk': 78.5
})
```

## Environment Variables

You can set defaults via environment variables in your shell:

```bash
# .bashrc or .zshrc
export MOUSTIQUE_HOST="moustique.example.com"
export MOUSTIQUE_PORT="33334"
export MOUSTIQUE_USER="alice"
export MOUSTIQUE_PASS="secret123"

# Then use CLI without specifying each time
./moustique-cli -a pub -t /topic -m "message"
```

**Note:** The current version doesn't read environment variables, but you can create a wrapper script:

```bash
#!/bin/bash
# Save as mq-cli
./moustique-cli -h ${MOUSTIQUE_HOST:-localhost} \
                -p ${MOUSTIQUE_PORT:-33334} \
                ${MOUSTIQUE_USER:+-u $MOUSTIQUE_USER} \
                ${MOUSTIQUE_PASS:+-pwd $MOUSTIQUE_PASS} \
                "$@"
```

## Tips and Tricks

### 1. Monitor Multiple Topics

```bash
# Open multiple terminals
terminal1$ ./moustique-cli -a sub -t /sensors/temp
terminal2$ ./moustique-cli -a sub -t /sensors/humidity
terminal3$ ./moustique-cli -a sub -t /alerts/#
```

### 2. Quick Testing

```bash
# Terminal 1: Subscribe
./moustique-cli -a sub -t /test

# Terminal 2: Publish
./moustique-cli -a pub -t /test -m "Hello!"
```

### 3. JSON Messages

```bash
# Store complex data
./moustique-cli -a put -t /config/app \
    -m '{"debug":true,"timeout":30,"retries":3}'

# Publish structured events
./moustique-cli -a pub -t /events/user_login \
    -m '{"user_id":123,"timestamp":"2025-12-24T12:00:00Z"}'
```

### 4. Debugging

```bash
# Use verbose mode to see client name
./moustique-cli -v -a pub -t /debug/test -m "checking"

# Subscribe with verbose to see all metadata
./moustique-cli -v -a sub -t /debug/#
```

## Troubleshooting

### Connection Issues

```bash
# Test connection
./moustique-cli -h localhost -p 33334 -a pub -t /ping -m "test"

# If authentication fails
./moustique-cli -u alice -pwd secret123 -a pub -t /test -m "auth test"
```

### Topic Naming Best Practices

```bash
# Good: Hierarchical structure
/sensors/room1/temperature
/logs/app/error
/events/user/login

# Avoid: Flat structure
/sensor_room1_temperature
/log_app_error
```

### Performance

```bash
# For high-frequency publishing, consider batching
for i in {1..100}; do
    ./moustique-cli -a pub -t /metrics -m "data$i"
done

# Better: Use a dedicated client library for high-frequency scenarios
```

## Getting Help

```bash
# Show help
./moustique-cli -help

# Check version
./moustique-cli -a version
```

## Next Steps

- See [README.md](README.md) for client library examples
- See [cmd/moustique-cli/README.md](cmd/moustique-cli/README.md) for detailed CLI documentation
- Check server logs for debugging: `/var/log/moustique.log` or server console output
