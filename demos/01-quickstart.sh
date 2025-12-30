#!/bin/bash
# Moustique Quick Start Demo
# Record with: asciinema rec quickstart.cast
# Convert to GIF: agg quickstart.cast quickstart.gif

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper function to type effect
type_effect() {
    text="$1"
    for (( i=0; i<${#text}; i++ )); do
        echo -n "${text:$i:1}"
        sleep 0.03
    done
    echo
}

# Helper function for pause
pause() {
    sleep ${1:-2}
}

clear
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}║           🦟 Moustique Message Broker                     ║${NC}"
echo -e "${BLUE}║           Quick Start Guide                               ║${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo
pause 2

echo -e "${GREEN}# Step 1: Start the server${NC}"
type_effect "./moustique"
echo
echo "Starting server on:"
echo "  HTTP: http://localhost:33334"
echo "  MQTT: tcp://localhost:1883"
pause 3

echo
echo -e "${GREEN}# Step 2: Subscribe to a topic (Terminal 1)${NC}"
type_effect "mosquitto_sub -h localhost -t 'sensors/temperature' -u demo -P demo123"
echo "Waiting for messages..."
pause 2

echo
echo -e "${GREEN}# Step 3: Publish a message (Terminal 2)${NC}"
type_effect "mosquitto_pub -h localhost -t 'sensors/temperature' -m '23.5°C' -u demo -P demo123"
pause 1

echo
echo -e "${YELLOW}📨 Message received in Terminal 1:${NC}"
echo "23.5°C"
pause 2

echo
echo -e "${GREEN}# You can also use HTTP API:${NC}"
type_effect "./moustique-cli -a pub -t 'sensors/humidity' -m '45%' -u demo -pwd demo123"
echo "✓ Published to sensors/humidity"
pause 2

echo
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}║   ✓ Moustique is running!                                 ║${NC}"
echo -e "${BLUE}║   • MQTT push messaging ⚡                                ║${NC}"
echo -e "${BLUE}║   • HTTP polling available 🌐                             ║${NC}"
echo -e "${BLUE}║   • Multi-tenant ready 🔐                                 ║${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}║   Open http://localhost:33334 for web UI                  ║${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo
