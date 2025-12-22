// clients/javascript/moustique/index.js

class Moustique {
    constructor({ ip = '192.168.1.79', port = '33334', clientName = '' } = {}) {
        this.ip = ip;
        this.port = port;
        this.baseUrl = `http://${ip}:${port}`;
        this.clientName = clientName || `${this._getHostname()}-${Math.floor(Math.random() * 100)}-${Date.now()}`;
        this.callbacks = new Map();
        this.systemCallbacks = new Map();
        this.systemCallbacks.set('/server/action/resubscribe', () => this.resubscribe());
    }

    _getHostname() {
        // Fungerar i browser (location.hostname) och Node (os.hostname fallback)
        if (typeof window !== 'undefined') return window.location?.hostname || 'browser';
        try { return require('os').hostname(); } catch { return 'node'; }
    }

    static enc(text) {
        if (!text) return '';
        const b64 = btoa(text);
        return b64.replace(/[A-Za-z]/g, c =>
            'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'.includes(c)
                ? 'NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm'['ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'.indexOf(c)]
                : c
        );
    }

    static dec(encoded) {
        if (!encoded) return '';
        const rotated = encoded.replace(/[A-Za-z]/g, c =>
            'NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm'['ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'.indexOf(c)]
        );
        return atob(rotated);
    }

    static getNiceDateTime() {
        return new Date().toISOString().replace('T', ' ').substring(0, 19);
    }

    async publish(topic, message) {
        const url = `${this.baseUrl}/POST`;
        const payload = {
            topic: Moustique.enc(topic),
            message: Moustique.enc(message),
            updated_time: Moustique.enc(Math.floor(Date.now() / 1000).toString()),
            updated_nicedatetime: Moustique.enc(Moustique.getNiceDateTime()),
            from: Moustique.enc(this.clientName)
        };

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
        const payload = {
            valname: Moustique.enc(topic),
            val: Moustique.enc(message),
            updated_time: Moustique.enc(Math.floor(Date.now() / 1000).toString()),
            updated_nicedatetime: Moustique.enc(Moustique.getNiceDateTime()),
            from: Moustique.enc(this.clientName)
        };

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
        const url = `${this.baseUrl}/SUBSCRIBE`;
        const payload = {
            topic: Moustique.enc(topic),
            client: Moustique.enc(this.clientName)
        };

        try {
            const res = await fetch(url, {
                method: 'POST',
                body: new URLSearchParams(payload)
            });
            if (res.ok) {
                console.log(`${this.clientName} subscribed to ${topic}`);
                if (!this.callbacks.has(topic)) this.callbacks.set(topic, []);
                this.callbacks.get(topic).push(callback);
            }
        } catch (err) {
            console.error('Subscribe error:', err);
        }
    }

    async pickup() {
        const url = `${this.baseUrl}/PICKUP`;
        const payload = { client: Moustique.enc(this.clientName) };

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
            await this.subscribe(topic, ...this.callbacks.get(topic));
        }
    }

    getClientName() {
        return this.clientName;
    }
}

export { Moustique };