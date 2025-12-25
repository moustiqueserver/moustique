#!/usr/bin/env python3
"""
Python client test for Moustique
Usage:
    python3 python_test.py public
    python3 python_test.py auth <username> <password>
"""

import sys
import os

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

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 python_test.py <public|auth> [username] [password]")
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
    else:
        print(f"Unknown mode: {mode}")
        sys.exit(1)

    sys.exit(0 if success else 1)
