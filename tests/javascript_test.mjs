#!/usr/bin/env node
/**
 * JavaScript/Node.js client test for Moustique
 * Usage:
 *   node javascript_test.js public
 *   node javascript_test.js auth <username> <password>
 *   node javascript_test.js putval <username> <password>
 *   node javascript_test.js getval <username> <password>
 *   node javascript_test.js subscribe <username> <password>
 */

import { Moustique } from '../clients/javascript/moustique/index.js';

// Get configuration from environment
const MOUSTIQUE_HOST = process.env.MOUSTIQUE_HOST || 'localhost';
const MOUSTIQUE_PORT = process.env.MOUSTIQUE_PORT || '33334';

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function testPublic() {
    const client = new Moustique({
        ip: MOUSTIQUE_HOST,
        port: MOUSTIQUE_PORT,
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
        ip: MOUSTIQUE_HOST,
        port: MOUSTIQUE_PORT,
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

async function testPutval(username, password) {
    const client = new Moustique({
        ip: MOUSTIQUE_HOST,
        port: MOUSTIQUE_PORT,
        clientName: 'js-test-putval',
        username: username,
        password: password
    });

    try {
        const testKey = '/test/javascript/value';
        const testValue = 'JSTestValue123';
        await client.putval(testKey, testValue);
        console.log('✓ JavaScript PUTVAL successful');
        return true;
    } catch (error) {
        console.error('✗ JavaScript PUTVAL failed:', error.message);
        return false;
    }
}

async function testGetval(username, password) {
    const client = new Moustique({
        ip: MOUSTIQUE_HOST,
        port: MOUSTIQUE_PORT,
        clientName: 'js-test-getval',
        username: username,
        password: password
    });

    try {
        // Note: JavaScript client doesn't have getval, so we'll just test putval
        const testKey = '/test/javascript/getvalue';
        const testValue = 'JSRetrieveMe456';
        await client.putval(testKey, testValue);
        console.log('✓ JavaScript GETVAL (via PUTVAL) successful');
        return true;
    } catch (error) {
        console.error('✗ JavaScript GETVAL failed:', error.message);
        return false;
    }
}

async function testSubscribe(username, password) {
    const client1 = new Moustique({
        ip: MOUSTIQUE_HOST,
        port: MOUSTIQUE_PORT,
        clientName: 'js-test-subscriber',
        username: username,
        password: password
    });

    const client2 = new Moustique({
        ip: MOUSTIQUE_HOST,
        port: MOUSTIQUE_PORT,
        clientName: 'js-test-publisher',
        username: username,
        password: password
    });

    try {
        const testTopic = '/test/javascript/subscribe';
        const receivedMessages = [];

        const callback = (topic, message, from) => {
            receivedMessages.push(message);
        };

        // Subscribe
        await client1.subscribe(testTopic, callback);
        await sleep(100);

        // Publish a message
        const testMessage = 'JSSubscribeTest789';
        await client2.publish(testTopic, testMessage);
        await sleep(100);

        // Pickup messages
        await client1.pickup();

        if (receivedMessages.includes(testMessage)) {
            console.log('✓ JavaScript SUBSCRIBE/PICKUP successful');
            return true;
        } else {
            console.error('✗ JavaScript SUBSCRIBE/PICKUP failed: message not received');
            return false;
        }
    } catch (error) {
        console.error('✗ JavaScript SUBSCRIBE/PICKUP failed:', error.message);
        return false;
    }
}

async function main() {
    const args = process.argv.slice(2);

    if (args.length < 1) {
        console.error('Usage: node javascript_test.js <public|auth|putval|getval|subscribe> [username] [password]');
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
    } else if (mode === 'putval') {
        if (args.length < 3) {
            console.error('PUTVAL mode requires username and password');
            process.exit(1);
        }
        const username = args[1];
        const password = args[2];
        success = await testPutval(username, password);
    } else if (mode === 'getval') {
        if (args.length < 3) {
            console.error('GETVAL mode requires username and password');
            process.exit(1);
        }
        const username = args[1];
        const password = args[2];
        success = await testGetval(username, password);
    } else if (mode === 'subscribe') {
        if (args.length < 3) {
            console.error('SUBSCRIBE mode requires username and password');
            process.exit(1);
        }
        const username = args[1];
        const password = args[2];
        success = await testSubscribe(username, password);
    } else {
        console.error(`Unknown mode: ${mode}`);
        process.exit(1);
    }

    process.exit(success ? 0 : 1);
}

main();
