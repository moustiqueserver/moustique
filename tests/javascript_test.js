#!/usr/bin/env node
/**
 * JavaScript/Node.js client test for Moustique
 * Usage:
 *   node javascript_test.js public
 *   node javascript_test.js auth <username> <password>
 */

const path = require('path');
const { Moustique } = require(path.join(__dirname, '..', 'clients', 'javascript', 'moustique'));

async function testPublic() {
    const client = new Moustique({
        ip: 'localhost',
        port: '33334',
        clientName: 'js-test-public'
    });

    try {
        await client.publish('/test/javascript/public', 'Hello from JavaScript public!');
        console.log('✓ JavaScript public publish successful');
        return true;
    } catch (error) {
        console.error('✗ JavaScript public publish failed:', error.message);
        return false;
    }
}

async function testAuth(username, password) {
    const client = new Moustique({
        ip: 'localhost',
        port: '33334',
        clientName: 'js-test-auth',
        username: username,
        password: password
    });

    try {
        await client.publish('/test/javascript/auth', 'Hello from JavaScript auth!');
        console.log('✓ JavaScript authenticated publish successful');
        return true;
    } catch (error) {
        console.error('✗ JavaScript authenticated publish failed:', error.message);
        return false;
    }
}

async function main() {
    const args = process.argv.slice(2);

    if (args.length < 1) {
        console.error('Usage: node javascript_test.js <public|auth> [username] [password]');
        process.exit(1);
    }

    const mode = args[0];
    let success;

    if (mode === 'public') {
        success = await testPublic();
    } else if (mode === 'auth') {
        if (args.length < 3) {
            console.error('Auth mode requires username and password');
            process.exit(1);
        }
        const username = args[1];
        const password = args[2];
        success = await testAuth(username, password);
    } else {
        console.error(`Unknown mode: ${mode}`);
        process.exit(1);
    }

    process.exit(success ? 0 : 1);
}

main();
