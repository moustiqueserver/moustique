#!/usr/bin/env python3
"""
Python client test for Moustique
Usage:
    python3 python_test.py public
    python3 python_test.py auth <username> <password>
    python3 python_test.py putval <username> <password>
    python3 python_test.py getval <username> <password>
    python3 python_test.py subscribe <username> <password>
"""

import sys
import os
import time

# Add parent directory to path to import client
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'clients', 'python'))

from moustique import Moustique

def test_public():
    """Test publishing to public broker"""
    client = Moustique(
        ip="localhost",
        port="33334",
        client_name="python-test-public"
    )

    try:
        client.publish("/test/python/public", "Hello from Python public!", "python-test")
        print("✓ Python public publish successful")
        return True
    except Exception as e:
        print(f"✗ Python public publish failed: {e}")
        return False

def test_auth(username, password):
    """Test publishing with authentication"""
    client = Moustique(
        ip="localhost",
        port="33334",
        client_name="python-test-auth",
        username=username,
        password=password
    )

    try:
        client.publish("/test/python/auth", "Hello from Python auth!", "python-test")
        print("✓ Python authenticated publish successful")
        return True
    except Exception as e:
        print(f"✗ Python authenticated publish failed: {e}")
        return False

def test_putval(username, password):
    """Test PUTVAL operation"""
    client = Moustique(
        ip="localhost",
        port="33334",
        client_name="python-test-putval",
        username=username,
        password=password
    )

    try:
        test_key = "/test/python/value"
        test_value = "TestValue123"
        client.putval(test_key, test_value, "python-test")
        print("✓ Python PUTVAL successful")
        return True
    except Exception as e:
        print(f"✗ Python PUTVAL failed: {e}")
        return False

def test_getval(username, password):
    """Test GETVAL operation"""
    client = Moustique(
        ip="localhost",
        port="33334",
        client_name="python-test-getval",
        username=username,
        password=password
    )

    try:
        # First put a value
        test_key = "/test/python/getvalue"
        test_value = "RetrieveMe456"
        client.putval(test_key, test_value, "python-test")
        time.sleep(0.1)  # Small delay

        # Then get it back
        retrieved = client.get_val(test_key)
        if retrieved == test_value:
            print("✓ Python GETVAL successful")
            return True
        else:
            print(f"✗ Python GETVAL failed: expected '{test_value}', got '{retrieved}'")
            return False
    except Exception as e:
        print(f"✗ Python GETVAL failed: {e}")
        return False

def test_subscribe(username, password):
    """Test SUBSCRIBE and PICKUP operations"""
    client1 = Moustique(
        ip="localhost",
        port="33334",
        client_name="python-test-subscriber",
        username=username,
        password=password
    )

    client2 = Moustique(
        ip="localhost",
        port="33334",
        client_name="python-test-publisher",
        username=username,
        password=password
    )

    try:
        test_topic = "/test/python/subscribe"
        received_messages = []

        def callback(topic, message, from_name):
            received_messages.append(message)

        # Subscribe
        client1.subscribe(test_topic, callback)
        time.sleep(0.1)

        # Publish a message
        test_message = "SubscribeTest789"
        client2.publish(test_topic, test_message, "python-test")
        time.sleep(0.1)

        # Pickup messages
        client1.pickup()

        if test_message in received_messages:
            print("✓ Python SUBSCRIBE/PICKUP successful")
            return True
        else:
            print(f"✗ Python SUBSCRIBE/PICKUP failed: message not received")
            return False
    except Exception as e:
        print(f"✗ Python SUBSCRIBE/PICKUP failed: {e}")
        return False

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 python_test.py <public|auth|putval|getval|subscribe> [username] [password]")
        sys.exit(1)

    mode = sys.argv[1]

    if mode == "public":
        success = test_public()
    elif mode == "auth":
        if len(sys.argv) < 4:
            print("Auth mode requires username and password")
            sys.exit(1)
        username = sys.argv[2]
        password = sys.argv[3]
        success = test_auth(username, password)
    elif mode == "putval":
        if len(sys.argv) < 4:
            print("PUTVAL mode requires username and password")
            sys.exit(1)
        username = sys.argv[2]
        password = sys.argv[3]
        success = test_putval(username, password)
    elif mode == "getval":
        if len(sys.argv) < 4:
            print("GETVAL mode requires username and password")
            sys.exit(1)
        username = sys.argv[2]
        password = sys.argv[3]
        success = test_getval(username, password)
    elif mode == "subscribe":
        if len(sys.argv) < 4:
            print("SUBSCRIBE mode requires username and password")
            sys.exit(1)
        username = sys.argv[2]
        password = sys.argv[3]
        success = test_subscribe(username, password)
    else:
        print(f"Unknown mode: {mode}")
        sys.exit(1)

    sys.exit(0 if success else 1)
