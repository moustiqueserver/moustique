#!/usr/bin/env python3
"""
Moustique Python Client – Multi-tenant Integration Test with HTTP and MQTT
"""

import sys
import time
from datetime import datetime
import os
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../moustique')))
from client import Moustique, getversion, getstats

def message_callback(topic: str, message: str, from_name: str):
    timestamp = datetime.now().strftime("%H:%M:%S")
    print(f"[{timestamp}] MESSAGE → '{topic}': {message} (from {from_name})")

def run_test(protocol: str, server_ip: str, server_port: str, username: str, password: str, mqtt_port: int = 1883):
    """Run test with specified protocol (http or mqtt)"""
    print(f"\n{'='*60}")
    print(f"Testing with {protocol.upper()} protocol")
    print(f"{'='*60}\n")

    use_mqtt = (protocol == "mqtt")

    print("=== Moustique Python Client – Multi-tenant Test ===")
    print(f"Protocol: {protocol.upper()}")
    print(f"Server: {server_ip}:{server_port}")
    if use_mqtt:
        print(f"MQTT Port: {mqtt_port}")
    print(f"Username: {username}\n")

    # Create client with authentication
    client = Moustique(
        ip=server_ip,
        port=server_port,
        client_name=f"TestRunner-{protocol}",
        username=username,
        password=password,
        use_mqtt=use_mqtt,
        mqtt_port=mqtt_port
    )

    print(f"Client ID: {client.get_client_name()}\n")

    try:
        # 1. Server version (no auth required)
        print("1. Getting server version...")
        version = getversion(server_ip, server_port)
        print(f"   → {version}\n")

        # 2. Publish
        print("2. Publishing message...")
        client.publish("/test/topic/python", f"Hello from {protocol.upper()} test!")
        time.sleep(0.5)

        # 3. Set value
        print("3. Setting value...")
        client.putval("/test/value/python", f"python-{protocol}-v1")
        time.sleep(0.5)

        # 4. Get value (only for HTTP, MQTT doesn't support getval)
        if not use_mqtt:
            print("4. Getting value...")
            value = client.get_val("/test/value/python")
            print(f"   → {value}\n")

        # 5. Subscribe and receive
        print(f"5. Subscribing to /test/topic/python...")
        client.subscribe("/test/topic/python", message_callback)

        print("   Sending message to trigger callback...")
        time.sleep(1)  # Give subscription time to register
        client.publish("/test/topic/python", f"This message should appear in callback! ({protocol.upper()})")

        if use_mqtt:
            print("   Waiting for MQTT messages (10 seconds)...")
            time.sleep(10)
        else:
            print("   Polling for HTTP messages (10 seconds)...")
            for i in range(20):
                client.tick()
                time.sleep(0.5)

        # 6. Statistics
        print("\n6. Getting statistics...")
        stats = getstats(server_ip, server_port, username, password)
        print(f"   Request count: {stats.get('request_count', 'N/A')}")
        print(f"   Active clients: {stats.get('clients', 'N/A')}")

        print(f"\n=== {protocol.upper()} test complete! ===")

    except Exception as e:
        print(f"\nError in {protocol.upper()} test: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        if use_mqtt:
            client.disconnect()

    return True

def main():
    if len(sys.argv) < 5:
        print("Usage: python test_client.py <ip> <port> <username> <password> [mqtt_port]")
        print("Example: python test_client.py localhost 33334 demo demo123")
        print("         python test_client.py localhost 33334 demo demo123 1883")
        sys.exit(1)

    server_ip = sys.argv[1]
    server_port = sys.argv[2]
    username = sys.argv[3]
    password = sys.argv[4]
    mqtt_port = int(sys.argv[5]) if len(sys.argv) > 5 else 1883

    print("\n" + "="*60)
    print("Moustique Python Client - HTTP + MQTT Integration Test")
    print("="*60)

    # Test HTTP
    http_success = run_test("http", server_ip, server_port, username, password)

    # Test MQTT
    mqtt_success = run_test("mqtt", server_ip, server_port, username, password, mqtt_port)

    # Summary
    print("\n" + "="*60)
    print("TEST SUMMARY")
    print("="*60)
    print(f"HTTP test: {'✓ PASSED' if http_success else '✗ FAILED'}")
    print(f"MQTT test: {'✓ PASSED' if mqtt_success else '✗ FAILED'}")
    print("="*60 + "\n")

    sys.exit(0 if (http_success and mqtt_success) else 1)

if __name__ == "__main__":
    main()
