#!/bin/bash
# Moustique Multi-Tenant Demo
# Record with: asciinema rec multitenant.cast

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
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
echo -e "${BLUE}║           Multi-Tenant Architecture                       ║${NC}"
echo -e "${BLUE}║           Isolated Brokers Per User                       ║${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo
pause 2

echo -e "${YELLOW}Scenario: Two companies using the same Moustique server${NC}"
echo
pause 1

echo -e "${GREEN}# Company A - Alice's IoT devices${NC}"
type_effect "mosquitto_pub -h localhost -t 'warehouse/temperature' -m '18.5°C' -u alice -P alice123"
echo "✓ Published to alice's broker"
pause 2

echo
echo -e "${BLUE}# Company B - Bob's IoT devices${NC}"
type_effect "mosquitto_pub -h localhost -t 'warehouse/temperature' -m '22.1°C' -u bob -P bob456"
echo "✓ Published to bob's broker"
pause 2

echo
echo "───────────────────────────────────────────────"
echo
pause 1

echo -e "${GREEN}# Alice subscribes (only sees HER messages)${NC}"
type_effect "mosquitto_sub -h localhost -t 'warehouse/#' -u alice -P alice123"
echo "18.5°C"
echo
pause 2

echo -e "${BLUE}# Bob subscribes (only sees HIS messages)${NC}"
type_effect "mosquitto_sub -h localhost -t 'warehouse/#' -u bob -P bob456"
echo "22.1°C"
echo
pause 2

echo -e "${RED}# Bob tries to access Alice's broker${NC}"
type_effect "mosquitto_sub -h localhost -t 'warehouse/#' -u bob -P alice123"
echo "❌ Connection Refused: Authentication failed"
pause 2

echo
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                   Isolation Guarantees                    ║${NC}"
echo -e "${BLUE}╠═══════════════════════════════════════════════════════════╣${NC}"
echo -e "${BLUE}║  ✓ Each user gets a separate broker instance             ║${NC}"
echo -e "${BLUE}║  ✓ Topics are completely isolated                        ║${NC}"
echo -e "${BLUE}║  ✓ No cross-tenant data leakage                          ║${NC}"
echo -e "${BLUE}║  ✓ Independent rate limits per user                      ║${NC}"
echo -e "${BLUE}║  ✓ Separate statistics and monitoring                    ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo
pause 3

echo -e "${YELLOW}Perfect for:${NC}"
echo "  • SaaS platforms"
echo "  • IoT device management companies"
echo "  • Multi-customer deployments"
echo "  • Shared hosting environments"
echo
