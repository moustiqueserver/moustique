#!/usr/bin/env python3
"""
Moustique Python Client – Integrationstest
Fungerar oavsett var du kör scriptet ifrån.
"""

import sys
import os
import time
from datetime import datetime

# === AUTOMATISK PATH-FIX – fungerar alltid ===
# Hitta projektroten och lägg till clients/python i sys.path
FILE = os.path.abspath(__file__)                          # t.ex. .../clients/python/tests/test_client.py
TEST_DIR = os.path.dirname(FILE)                          # .../clients/python/tests
PYTHON_CLIENT_DIR = os.path.dirname(TEST_DIR)             # .../clients/python
PROJECT_ROOT = os.path.dirname(PYTHON_CLIENT_DIR)         # .../Source/Moustique

# Lägg till clients/python så att 'moustique' blir importerbart
if PYTHON_CLIENT_DIR not in sys.path:
    sys.path.insert(0, PYTHON_CLIENT_DIR)

# Nu fungerar importen!
from moustique import Moustique
from moustique import (
    getversion, getstats, getclients, gettopics,
    getposters, getpeerhosts, getcrooks
)

def message_callback(topic: str, message: str, from_name: str):
    timestamp = datetime.now().strftime("%H:%M:%S")
    print(f"[{timestamp}] MEDDELANDE → '{topic}': {message} (från {from_name})")

def main():
    if len(sys.argv) < 3:
        print("Användning: python clients/python/tests/test_client.py <ip> <port> [lösenord]")
        print("Exempel: python clients/python/tests/test_client.py 192.168.1.79 33334 1Delmataren")
        sys.exit(1)

    server_ip = sys.argv[1]
    server_port = sys.argv[2]
    password = sys.argv[3] if len(sys.argv) > 3 else ""

    print("=== Moustique Python Client – Integrationstest ===")
    print(f"Ansluter till: http://{server_ip}:{server_port}")
    print(f"Lösenord: {'angett' if password else 'inget (hoppas på öppet)'}\n")

    client = Moustique(ip=server_ip, port=server_port, client_name="TestRunner")
    print(f"Klient-ID: {client.get_client_name()}\n")

    try:
        # 1. Serverversion
        print("1. Hämtar serverversion...")
        version = getversion(server_ip, server_port, password)
        print(f"   → {version}\n")

        # 2. Publicera
        print("2. Publicerar meddelande...")
        client.publish("/test/topic", "Hej från det nya testscriptet!")
        time.sleep(1)

        # 3. PUTVAL
        print("3. Sätter värde...")
        client.putval("/test/value", "python-client-v2")
        time.sleep(1)

        # 4. GETVAL
        print("4. Hämtar värde...")
        value = client.get_val("/test/value")
        print(f"   → {value}\n")

        # 5. Prenumerera + ta emot
        print("5. Prenumererar på /test/topic...")
        client.subscribe("/test/topic", message_callback)

        print("   Skickar meddelande till sig själv...")
        client.publish("/test/topic", "Detta borde dyka upp i callbacken nedan!")

        print("   Pickup-loop i 10 sekunder...")
        for _ in range(20):
            client.tick()
            time.sleep(0.5)

        # 6. Resubscribe
        print("\n6. Testar resubscribe...")
        client.resubscribe()
        time.sleep(1)

        # 7. Statistik
        print("\n7. Hämtar statistik...")
        stats = getstats(server_ip, server_port, password)
        print(f"   → {stats}")

        clients_info = getclients(server_ip, server_port, password)
        print(f"   Aktiva klienter: {clients_info}")

        print("\n=== Test klart! ===")
        print("Klienten lyssnar vidare – avsluta med Ctrl+C")

        try:
            while True:
                client.tick()
                time.sleep(1)
        except KeyboardInterrupt:
            print("\n\nAvslutar testklienten.")
            sys.exit(0)

    except Exception as e:
        print(f"\nFel: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()