#!/bin/bash
# Script to create test user for Moustique tests
# This uses the server's API to create a test user

HOST="${MOUSTIQUE_HOST:-localhost}"
PORT="${MOUSTIQUE_PORT:-33334}"
TEST_USER="${TEST_USER:-testuser}"
TEST_PASS="${TEST_PASS:-testpass123}"
ADMIN_PASS="${ADMIN_PASS:-admin}"  # Default admin password

echo "Setting up test user for Moustique..."
echo "Host: $HOST:$PORT"
echo "Test user: $TEST_USER"
echo ""

# Note: This requires the admin password to be set
# In a production setup, you would need to configure this properly

echo "To create the test user, please:"
echo "1. Open http://$HOST:$PORT/superadmin in your browser"
echo "2. Create a user with:"
echo "   - Username: $TEST_USER"
echo "   - Password: $TEST_PASS"
echo ""
echo "Or use the Python script to create the user programmatically."

# TODO: Add automated user creation via API when available
