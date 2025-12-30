// clients/javascript/tests/test_client.js
// Moustique JavaScript Client - HTTP + MQTT Integration Test

import { Moustique } from '../moustique/index.js';

const callback = (topic, message, from) => {
    const timestamp = new Date().toLocaleTimeString();
    console.log(`[${timestamp}] MESSAGE → '${topic}': ${message} (from ${from})`);
};

async function runTest(protocol, ip, port, username, password, mqttPort = 1883) {
    console.log('\n' + '='.repeat(60));
    console.log(`Testing with ${protocol.toUpperCase()} protocol`);
    console.log('='.repeat(60) + '\n');

    const useMqtt = (protocol === 'mqtt');

    console.log('=== Moustique JavaScript Client – Multi-tenant Test ===');
    console.log(`Protocol: ${protocol.toUpperCase()}`);
    console.log(`Server: ${ip}:${port}`);
    if (useMqtt) {
        console.log(`MQTT Port: ${mqttPort}`);
    }
    console.log(`Username: ${username}\n`);

    const client = new Moustique({
        ip,
        port,
        clientName: `JS-TestClient-${protocol}`,
        username,
        password,
        useMqtt,
        mqttPort
    });

    console.log(`Client ID: ${client.getClientName()}\n`);

    try {
        // 1. Publish
        console.log('1. Publishing message...');
        await client.publish('/test/topic/javascript', `Hello from ${protocol.toUpperCase()} test!`);
        await new Promise(resolve => setTimeout(resolve, 500));

        // 2. Set value
        console.log('2. Setting value...');
        await client.putval('/test/value/javascript', `js-${protocol}-v1`);
        await new Promise(resolve => setTimeout(resolve, 500));

        // 3. Subscribe and receive
        console.log('3. Subscribing to /test/topic/javascript...');
        await client.subscribe('/test/topic/javascript', callback);

        console.log('   Sending message to trigger callback...');
        await new Promise(resolve => setTimeout(resolve, 1000)); // Give subscription time
        await client.publish('/test/topic/javascript', `This message should appear in callback! (${protocol.toUpperCase()})`);

        if (useMqtt) {
            console.log('   Waiting for MQTT messages (10 seconds)...');
            await new Promise(resolve => setTimeout(resolve, 10000));
        } else {
            console.log('   Polling for HTTP messages (10 seconds)...');
            const interval = setInterval(() => client.pickup(), 500);
            await new Promise(resolve => setTimeout(resolve, 10000));
            clearInterval(interval);
        }

        console.log(`\n=== ${protocol.toUpperCase()} test complete! ===`);

        if (useMqtt) {
            client.disconnect();
        }

        return true;

    } catch (error) {
        console.error(`\nError in ${protocol.toUpperCase()} test:`, error);
        return false;
    }
}

async function main() {
    const args = process.argv.slice(2);

    if (args.length < 4) {
        console.log('Usage: node test_client.js <ip> <port> <username> <password> [mqtt_port]');
        console.log('Example: node test_client.js localhost 33334 demo demo123');
        console.log('         node test_client.js localhost 33334 demo demo123 1883');
        process.exit(1);
    }

    const [ip, port, username, password] = args;
    const mqttPort = args[4] ? parseInt(args[4]) : 1883;

    console.log('\n' + '='.repeat(60));
    console.log('Moustique JavaScript Client - HTTP + MQTT Integration Test');
    console.log('='.repeat(60));

    // Test HTTP
    const httpSuccess = await runTest('http', ip, port, username, password);

    // Test MQTT
    const mqttSuccess = await runTest('mqtt', ip, port, username, password, mqttPort);

    // Summary
    console.log('\n' + '='.repeat(60));
    console.log('TEST SUMMARY');
    console.log('='.repeat(60));
    console.log(`HTTP test: ${httpSuccess ? '✓ PASSED' : '✗ FAILED'}`);
    console.log(`MQTT test: ${mqttSuccess ? '✓ PASSED' : '✗ FAILED'}`);
    console.log('='.repeat(60) + '\n');

    process.exit((httpSuccess && mqttSuccess) ? 0 : 1);
}

main();