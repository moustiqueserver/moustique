#!/usr/bin/env python3
"""
Moustique Python Client – Multi-tenant Integration Test
"""

import sys
import time
from datetime import datetime
import sys
import os
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../moustique')))
from client import Moustique, getversion, getstats

def message_callback(topic: str, message: str, from_name: str):
    timestamp = datetime.now().strftime("%H:%M:%S")
    print(f"[{timestamp}] MESSAGE → '{topic}': {message} (from {from_name})")

def main():
    if len(sys.argv) < 5:
        print("Usage: python test_client.py <ip> <port> <username> <password>")
        print("Example: python test_client.py localhost 33334 demo demo123")
        sys.exit(1)

    server_ip = sys.argv[1]
    server_port = sys.argv[2]
    username = sys.argv[3]
    password = sys.argv[4]

    print("=== Moustique Python Client – Multi-tenant Test ===")
    print(f"Connecting to: http://{server_ip}:{server_port}")
    print(f"Username: {username}\n")

    # Create client with authentication
    client = Moustique(
        ip=server_ip,
        port=server_port,
        client_name="TestRunner",
        username=username,
        password=password
    )

    print(f"Client ID: {client.get_client_name()}\n")

    try:
        # 1. Server version (no auth required)
        print("1. Getting server version...")
        #version = getversion(server_ip, server_port)
        #, username, password)
        #print(f"   → {version}\n")

        # 2. Publish
        print("2. Publishing message...")
        client.publish("/test/topic", "Hello from multi-tenant test!",username)
        #, password)
        time.sleep(0.5)

        # 3. Set value
        print("3. Setting value...")
        client.putval("/test/value", "python-multitenant-v1", username)
        #, password)
        time.sleep(0.5)

        # 4. Get value
        print("4. Getting value...")
        value = client.get_val("/test/value")
        #, username, password)
        print(f"   → {value}\n")

        # 5. Subscribe and receive
        print("5. Subscribing to /test/topic...")
        client.subscribe("/test/topic", message_callback)
        #, username, password)

        print("   Sending message to trigger callback...")
        client.publish("/test/topic", "This message should appear in callback!")

        print("   Polling for 10 seconds...")
        for i in range(20):
            client.tick()
            time.sleep(0.5)

        # 6. Statistics
        print("\n6. Getting statistics...")
        stats = getstats(server_ip, server_port, username, password)
        print(f"   Request count: {stats.get('request_count', 'N/A')}")
        print(f"   Active clients: {stats.get('clients', 'N/A')}")

        print("\n=== Test complete! ===")
        print("Press Ctrl+C to exit")

        try:
            while True:
                client.tick()
                time.sleep(1)
        except KeyboardInterrupt:
            print("\n\nExiting test client.")
            sys.exit(0)

    except Exception as e:
        print(f"\nError: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()
