# Moustique Demo Scripts

This directory contains demo scripts that showcase Moustique's key features. These scripts can be used to create GIF animations for documentation and promotional materials.

## Prerequisites

```bash
# Install asciinema for terminal recording
sudo apt-get install asciinema   # Ubuntu/Debian
brew install asciinema           # macOS

# Install agg for converting to GIF (optional)
cargo install --git https://github.com/asciinema/agg
```

## Available Demos

### 1. Quick Start (`01-quickstart.sh`)

Shows the absolute basics of getting started with Moustique:
- Starting the server
- Publishing with MQTT
- Subscribing with mosquitto tools
- Using the CLI

**Run:**
```bash
chmod +x demos/01-quickstart.sh
asciinema rec demos/quickstart.cast -c "demos/01-quickstart.sh"
```

**Convert to GIF:**
```bash
agg demos/quickstart.cast demos/quickstart.gif
```

### 2. MQTT vs HTTP (`02-mqtt-vs-http.sh`)

Demonstrates the dual-protocol advantage:
- MQTT real-time push
- HTTP polling fallback
- Performance comparison
- When to use each protocol

**Run:**
```bash
chmod +x demos/02-mqtt-vs-http.sh
asciinema rec demos/mqtt-vs-http.cast -c "demos/02-mqtt-vs-http.sh"
```

### 3. Multi-Tenant (`03-multitenant.sh`)

Shows how tenant isolation works:
- Separate brokers per user
- No cross-tenant data leakage
- Authentication and authorization
- Use cases

**Run:**
```bash
chmod +x demos/03-multitenant.sh
asciinema rec demos/multitenant.cast -c "demos/03-multitenant.sh"
```

## Creating Custom Demos

To create your own demo:

1. Copy one of the existing scripts as a template
2. Modify the content to showcase your feature
3. Use the helper functions:
   - `type_effect "command"` - Types text with realistic speed
   - `pause 2` - Pause for 2 seconds
   - Color variables: `$GREEN`, `$BLUE`, `$YELLOW`, `$RED`, `$NC`

## Tips for Good Animations

- **Keep it short**: 30-60 seconds max
- **Focus on one concept**: Don't try to show everything
- **Use pauses**: Give viewers time to read
- **Clear visuals**: Use colors and boxes for emphasis
- **Show results**: Always demonstrate the output

## Publishing

Once you've created your GIFs, add them to the README:

```markdown
![Quick Start](demos/quickstart.gif)
```

Or use them in blog posts, tweets, and documentation.

## Live Demos

You can also share the `.cast` files directly:
- Upload to asciinema.org for web playback
- Embed in documentation with the asciinema player
- Share links on social media

Example:
```bash
asciinema upload demos/quickstart.cast
```

## Automated Recording

For CI/CD or batch generation:

```bash
#!/bin/bash
for script in demos/*.sh; do
    basename=$(basename "$script" .sh)
    asciinema rec "demos/${basename}.cast" -c "$script" --overwrite
    agg "demos/${basename}.cast" "demos/${basename}.gif"
done
```
