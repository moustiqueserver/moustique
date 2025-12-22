// clients/javascript/tests/test_client.js
import { Moustique } from '../moustique/index.js';

const client = new Moustique({
    ip: '127.0.0.1',
    port: '33335',
    clientName: 'JS-TestClient'
});

console.log('Klientnamn:', client.getClientName());

const callback = (topic, message, from) => {
    console.log(`[JS] Meddelande på ${topic}: ${message} (från ${from})`);
};

async function runTests() {
    await client.publish('/test/topic', 'Hej från JavaScript-klienten!');
    await client.putval('/test/value', 'js-value-42');
    await client.subscribe('/test/topic', callback);
    await client.publish('/test/topic', 'Detta triggar callback i JS!');

    // Poll i 10 sekunder
    console.log('Lyssnar i 10 sekunder...');
    const interval = setInterval(() => client.pickup(), 1000);
    setTimeout(() => {
        clearInterval(interval);
        console.log('Test klart!');
        process.exit(0);
    }, 10000);
}

runTests();