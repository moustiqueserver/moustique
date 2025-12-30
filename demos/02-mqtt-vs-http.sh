#!/bin/bash
# Moustique MQTT vs HTTP Demo
# Record with: asciinema rec mqtt-vs-http.cast
# Convert to GIF: agg mqtt-vs-http.cast mqtt-vs-http.gif

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
MAGENTA='\033[0;35m'
NC='\033[0m'

type_effect() {
    text="$1"
    for (( i=0; i<${#text}; i++ )); do
        echo -n "${text:$i:1}"
        sleep 0.03
    done
    echo
}

pause() {
    sleep ${1:-2}
}

clear
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}║              MQTT vs HTTP: Choose Your Protocol           ║${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo
pause 2

echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${MAGENTA}  Scenario: IoT Temperature Sensor                        ${NC}"
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
pause 2

echo -e "${GREEN}Option 1: MQTT (Push - Real-time)${NC}"
echo "────────────────────────────────────"
type_effect "python3 << EOF"
type_effect "from moustique import Moustique"
type_effect ""
type_effect "client = Moustique("
type_effect "    ip='localhost', port='33334',"
type_effect "    use_mqtt=True,  # ⚡ Real-time push"
type_effect "    username='demo', password='demo123'"
type_effect ")"
type_effect ""
type_effect "def on_temp(topic, msg, from_):"
type_effect "    print(f'🌡️  {msg}')"
type_effect ""
type_effect "client.subscribe('sensors/temperature', on_temp)"
type_effect "EOF"
echo
echo "✓ Connected via MQTT"
echo "✓ Messages delivered instantly (no polling)"
echo "✓ Low latency: <5ms"
pause 3

echo
echo -e "${YELLOW}Option 2: HTTP (Poll - Universal)${NC}"
echo "────────────────────────────────────"
type_effect "python3 << EOF"
type_effect "from moustique import Moustique"
type_effect "import time"
type_effect ""
type_effect "client = Moustique("
type_effect "    ip='localhost', port='33334',"
type_effect "    use_mqtt=False,  # 🌐 HTTP polling"
type_effect "    username='demo', password='demo123'"
type_effect ")"
type_effect ""
type_effect "client.subscribe('sensors/temperature', on_temp)"
type_effect ""
type_effect "while True:"
type_effect "    client.tick()  # Poll for messages"
type_effect "    time.sleep(1)"
type_effect "EOF"
echo
echo "✓ Works everywhere (firewalls, proxies)"
echo "✓ Simple REST API"
echo "✓ Latency: ~1000ms (poll interval)"
pause 3

echo
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                   Performance Comparison                  ║${NC}"
echo -e "${BLUE}╠═══════════════════════════════════════════════════════════╣${NC}"
echo -e "${BLUE}║  Metric         │  MQTT          │  HTTP                  ║${NC}"
echo -e "${BLUE}║─────────────────┼────────────────┼────────────────────────║${NC}"
echo -e "${BLUE}║  Latency        │  <5ms          │  ~1000ms (polling)     ║${NC}"
echo -e "${BLUE}║  Bandwidth      │  Low           │  Medium                ║${NC}"
echo -e "${BLUE}║  Firewall       │  May be blocked│  Works everywhere      ║${NC}"
echo -e "${BLUE}║  Battery        │  Efficient     │  More drain (polling)  ║${NC}"
echo -e "${BLUE}║  Real-time      │  Yes ⚡        │  No (delayed)          ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo
pause 3

echo -e "${GREEN}💡 Pro Tip: Use MQTT when possible, HTTP as fallback${NC}"
echo
echo "Moustique clients auto-detect and use the best available:"
echo "  1. Try MQTT first (fast, efficient)"
echo "  2. Fall back to HTTP if MQTT unavailable"
echo
pause 2

echo -e "${YELLOW}Both protocols work together seamlessly!${NC}"
echo "  • MQTT client can receive messages from HTTP publisher"
echo "  • HTTP client can receive messages from MQTT publisher"
echo "  • Mix and match as needed!"
echo
pause 2
