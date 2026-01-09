// clients/javascript/moustique/index.js

// Optional MQTT support
let mqtt = null;
let MQTT_AVAILABLE = false;
try {
    mqtt = await import('mqtt');
    MQTT_AVAILABLE = true;
} catch {
    // MQTT not available, will use HTTP only
}

class Moustique {
    constructor({ ip = '127.0.0.1', port = '33335', clientName = '', username = null, password = null,
                  useMqtt = false, mqttPort = 1883, useTls = false } = {}) {
        this.ip = ip;
        this.port = port;
        const scheme = useTls ? 'https' : 'http';
        this.baseUrl = `${scheme}://${ip}:${port}`;
        this.clientName = clientName || `${this._getHostname()}-${Math.floor(Math.random() * 100)}-${Date.now()}`;
        this.username = username;
        this.password = password;
        this.callbacks = new Map();
        this.systemCallbacks = new Map();
        this.systemCallbacks.set('/server/action/resubscribe', () => this.resubscribe());

        // MQTT support
        this.useMqtt = useMqtt && MQTT_AVAILABLE;
        this.mqttPort = mqttPort;
        this.mqttClient = null;
        this.mqttConnected = false;

        if (useMqtt && !MQTT_AVAILABLE) {
            console.warn('MQTT requested but mqtt package not installed. Falling back to HTTP.');
            console.warn('Install with: npm install mqtt');
        }

        if (this.useMqtt) {
            this._initMqtt();
        }
    }

    _getHostname() {
        // Fungerar i browser (location.hostname) och Node (os.hostname fallback)
        if (typeof window !== 'undefined') return window.location?.hostname || 'browser';
        try { return require('os').hostname(); } catch { return 'node'; }
    }

    static enc(text) {
        if (!text) return '';
        // Must match server encoding: ROT13 first, then Base64
        const rotated = text.replace(/[A-Za-z]/g, c =>
            'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'.includes(c)
                ? 'NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm'['ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'.indexOf(c)]
                : c
        );
        return btoa(rotated);
    }

    static dec(encoded) {
        if (!encoded) return '';
        // Reverse of encode: Base64 decode first, then ROT13
        const decoded = atob(encoded);
        return decoded.replace(/[A-Za-z]/g, c =>
            'NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm'['ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'.indexOf(c)]
        );
    }

    static getNiceDateTime() {
        return new Date().toISOString().replace('T', ' ').substring(0, 19);
    }

    _addAuth(payload) {
        if (this.username && this.password) {
            payload.username = Moustique.enc(this.username);
            payload.password = Moustique.enc(this.password);
        }
        return payload;
    }

    async publish(topic, message) {
        const url = `${this.baseUrl}/POST`;
        const payload = this._addAuth({
            topic: Moustique.enc(topic),
            message: Moustique.enc(message),
            updated_time: Moustique.enc(Math.floor(Date.now() / 1000).toString()),
            updated_nicedatetime: Moustique.enc(Moustique.getNiceDateTime()),
            from: Moustique.enc(this.clientName)
        });

        try {
            const res = await fetch(url, {
                method: 'POST',
                body: new URLSearchParams(payload),
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
            });
            if (res.ok) {
                console.log(`Published to ${topic}`);
            } else {
                console.error(`Publish failed: ${res.status} ${await res.text()}`);
            }
        } catch (err) {
            console.error('Publish error:', err);
        }
    }

    async putval(topic, message) {
        const url = `${this.baseUrl}/PUTVAL`;
        const payload = this._addAuth({
            valname: Moustique.enc(topic),
            val: Moustique.enc(message),
            updated_time: Moustique.enc(Math.floor(Date.now() / 1000).toString()),
            updated_nicedatetime: Moustique.enc(Moustique.getNiceDateTime()),
            from: Moustique.enc(this.clientName)
        });

        try {
            const res = await fetch(url, {
                method: 'PUT',
                body: new URLSearchParams(payload),
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
            });
            if (res.ok || res.status === 308) {
                console.log(`Putval ${topic} = ${message}`);
            } else {
                console.error(`Putval failed: ${res.status}`);
            }
        } catch (err) {
            console.error('Putval error:', err);
        }
    }

    async subscribe(topic, callback) {
        if (!this.callbacks.has(topic)) this.callbacks.set(topic, []);
        this.callbacks.get(topic).push(callback);

        if (this.useMqtt && this.mqttConnected) {
            // MQTT subscription
            try {
                this.mqttClient.subscribe(topic);
                console.log(`✓ Subscribed to ${topic} via MQTT`);
                return;
            } catch (err) {
                console.error('MQTT subscribe failed, falling back to HTTP:', err);
                // Fall through to HTTP subscription
            }
        }

        // HTTP subscription
        const url = `${this.baseUrl}/SUBSCRIBE`;
        const payload = this._addAuth({
            topic: Moustique.enc(topic),
            client: Moustique.enc(this.clientName)
        });

        try {
            const res = await fetch(url, {
                method: 'POST',
                body: new URLSearchParams(payload)
            });
            if (res.ok) {
                console.log(`${this.clientName} subscribed to ${topic}`);
            }
        } catch (err) {
            console.error('Subscribe error:', err);
        }
    }

    async pickup() {
        const url = `${this.baseUrl}/PICKUP`;
        const payload = this._addAuth({ client: Moustique.enc(this.clientName) });

        try {
            const res = await fetch(url, {
                method: 'POST',
                body: new URLSearchParams(payload)
            });
            if (!res.ok) return;

            const encrypted = await res.text();
            const decrypted = Moustique.dec(encrypted.trim());
            const data = decrypted ? JSON.parse(decrypted) : {};

            for (const [topic, messages] of Object.entries(data)) {
                for (const msg of messages) {
                    const callbacks = this.callbacks.get(topic) || [];
                    for (const cb of callbacks) {
                        cb(msg.topic, msg.message, msg.from);
                    }
                    const sysCb = this.systemCallbacks.get(topic);
                    if (sysCb && callbacks.length === 0) {
                        sysCb(msg.topic, msg.message);
                    }
                }
            }
        } catch (err) {
            console.error('Pickup error:', err);
        }
    }

    async resubscribe() {
        console.log(`${this.clientName} resubscribing...`);
        for (const topic of this.callbacks.keys()) {
            if (this.useMqtt && this.mqttConnected) {
                // MQTT resubscribe
                try {
                    this.mqttClient.subscribe(topic);
                    console.log(`✓ Re-subscribed to ${topic} via MQTT`);
                } catch (err) {
                    console.error(`Failed to resubscribe to '${topic}' via MQTT:`, err);
                }
            } else {
                // HTTP resubscribe
                const payload = this._addAuth({
                    topic: Moustique.enc(topic),
                    client: Moustique.enc(this.clientName)
                });

                try {
                    const res = await fetch(`${this.baseUrl}/SUBSCRIBE`, {
                        method: 'POST',
                        body: new URLSearchParams(payload)
                    });
                    if (res.ok) {
                        console.log(`✓ Re-subscribed to ${topic}`);
                    }
                } catch (err) {
                    console.error(`Failed to resubscribe to '${topic}':`, err);
                }
            }
        }
    }

    getClientName() {
        return this.clientName;
    }

    /**
     * Set the client's "AboutMe" description.
     * This description can be viewed in the admin panel and helps identify
     * what this client does (e.g., "Cron job on server X that sends sensor data").
     * @param {string} aboutMe - Description of what this client does
     */
    async setAboutMe(aboutMe) {
        const url = `${this.baseUrl}/SET_ABOUT_ME`;
        const payload = this._addAuth({
            client: Moustique.enc(this.clientName),
            about_me: Moustique.enc(aboutMe),
            type: Moustique.enc('client')
        });

        try {
            const res = await fetch(url, {
                method: 'POST',
                body: new URLSearchParams(payload),
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
            });
            if (!res.ok) {
                console.error(`setAboutMe failed: ${res.status} ${await res.text()}`);
            }
        } catch (err) {
            console.error('setAboutMe error:', err);
        }
    }

    // MQTT Support Methods

    _initMqtt() {
        try {
            const brokerUrl = `mqtt://${this.ip}:${this.mqttPort}`;
            const options = {
                clientId: this.clientName,
                clean: true
            };

            if (this.username && this.password) {
                options.username = this.username;
                options.password = this.password;
            }

            this.mqttClient = mqtt.connect(brokerUrl, options);

            this.mqttClient.on('connect', () => {
                this.mqttConnected = true;
                console.log(`✓ Connected to MQTT broker at ${brokerUrl}`);

                // Resubscribe to all topics
                for (const topic of this.callbacks.keys()) {
                    this.mqttClient.subscribe(topic);
                    console.log(`✓ Subscribed to ${topic} via MQTT`);
                }
            });

            this.mqttClient.on('message', (topic, payload) => {
                try {
                    const payloadStr = payload.toString();
                    let msgTopic, msgText, msgFrom;

                    // Try to parse as JSON first (for compatibility)
                    // If it fails, treat as plaintext (standard MQTT)
                    try {
                        const msgObj = JSON.parse(payloadStr);
                        msgTopic = msgObj.topic || topic;
                        msgText = msgObj.message || '';
                        msgFrom = msgObj.from || '';
                    } catch {
                        // Standard MQTT: plaintext payload
                        msgTopic = topic;
                        msgText = payloadStr;
                        msgFrom = '';
                    }

                    // Find matching callbacks
                    for (const [subscribedTopic, callbacks] of this.callbacks.entries()) {
                        if (this._topicMatches(subscribedTopic, msgTopic)) {
                            for (const callback of callbacks) {
                                try {
                                    callback(msgTopic, msgText, msgFrom);
                                } catch (err) {
                                    console.error(`Error in callback for topic '${topic}':`, err);
                                }
                            }
                        }
                    }
                } catch (err) {
                    console.error('Error processing MQTT message:', err);
                }
            });

            this.mqttClient.on('close', () => {
                this.mqttConnected = false;
                console.warn('⚠️  MQTT connection closed');
            });

            this.mqttClient.on('error', (err) => {
                console.error('MQTT error:', err);
                this.useMqtt = false;
            });

        } catch (err) {
            console.error('MQTT connection failed:', err);
            console.error('Falling back to HTTP mode');
            this.useMqtt = false;
        }
    }

    _topicMatches(pattern, topic) {
        // Simple MQTT wildcard matching (+ for single level, # for multi-level)
        const patternParts = pattern.split('/');
        const topicParts = topic.split('/');

        if (patternParts.length > topicParts.length && patternParts[patternParts.length - 1] !== '#') {
            return false;
        }

        for (let i = 0; i < patternParts.length; i++) {
            if (patternParts[i] === '#') {
                return true; // Match everything after
            }
            if (i >= topicParts.length) {
                return false;
            }
            if (patternParts[i] === '+') {
                continue; // Match single level
            }
            if (patternParts[i] !== topicParts[i]) {
                return false;
            }
        }

        return patternParts.length === topicParts.length || patternParts[patternParts.length - 1] === '#';
    }

    disconnect() {
        if (this.mqttClient && this.mqttConnected) {
            this.mqttClient.end();
            this.mqttConnected = false;
            console.log('MQTT client disconnected');
        }
    }
}

export { Moustique };