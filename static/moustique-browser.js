/**
 * Moustique Browser Client
 * Browser-compatible version of the Moustique JavaScript client
 *
 * Usage:
 *   <script src="http://your-server:33334/moustique.js"></script>
 *   <script>
 *     const client = new Moustique({
 *       ip: 'localhost',
 *       port: '33334',
 *       username: 'demo',
 *       password: 'demo123'
 *     });
 *
 *     client.subscribe('/sensors/#', function(topic, message, from) {
 *       console.log('Got:', topic, message);
 *     });
 *
 *     client.publish('/sensors/temp', '23.5');
 *
 *     // Start polling for messages
 *     setInterval(function() { client.pickup(); }, 1000);
 *   </script>
 */

(function(global) {
    'use strict';

    function Moustique(options) {
        options = options || {};
        this.ip = options.ip || 'localhost';
        this.port = options.port || '33334';
        this.baseUrl = (options.protocol || 'http:') + '//' + this.ip + ':' + this.port;
        this.clientName = options.clientName || ('browser-' + Math.random().toString(36).substring(2, 9));
        this.username = options.username || null;
        this.password = options.password || null;
        this.callbacks = new Map();
        this.debug = options.debug || false;
    }

    // ROT13 encoding (matching the official client)
    Moustique.prototype.rot13 = function(str) {
        return str.replace(/[A-Za-z]/g, function(c) {
            var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
            var rot = 'NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm';
            return rot[chars.indexOf(c)];
        });
    };

    // Encode for Moustique API: ROT13 first, then Base64
    Moustique.prototype.enc = function(text) {
        if (!text) return '';
        return btoa(this.rot13(String(text)));
    };

    // Decode from Moustique API: Base64 first, then ROT13
    Moustique.prototype.dec = function(encoded) {
        if (!encoded) return '';
        try {
            return this.rot13(atob(encoded));
        } catch (e) {
            return encoded;
        }
    };

    Moustique.prototype.getNiceDateTime = function() {
        return new Date().toISOString().replace('T', ' ').substring(0, 19);
    };

    // Subscribe to a topic
    Moustique.prototype.subscribe = function(topic, callback) {
        var self = this;

        if (!this.callbacks.has(topic)) {
            this.callbacks.set(topic, []);
        }
        this.callbacks.get(topic).push(callback);

        var payload = new URLSearchParams({
            topic: this.enc(topic),
            client: this.enc(this.clientName)
        });

        if (this.username && this.password) {
            payload.append('username', this.enc(this.username));
            payload.append('password', this.enc(this.password));
        }

        return fetch(this.baseUrl + '/SUBSCRIBE', {
            method: 'POST',
            body: payload,
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
        }).then(function(res) {
            if (res.ok) {
                if (self.debug) console.log('Moustique: Subscribed to ' + topic);
                return true;
            } else {
                console.error('Moustique: Subscribe failed:', res.status);
                return false;
            }
        }).catch(function(err) {
            console.error('Moustique: Subscribe error:', err);
            return false;
        });
    };

    // Pickup messages (poll)
    Moustique.prototype.pickup = function() {
        var self = this;

        var payload = new URLSearchParams({
            client: this.enc(this.clientName)
        });

        if (this.username && this.password) {
            payload.append('username', this.enc(this.username));
            payload.append('password', this.enc(this.password));
        }

        return fetch(this.baseUrl + '/PICKUP', {
            method: 'POST',
            body: payload,
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
        }).then(function(res) {
            if (!res.ok) return;
            return res.text();
        }).then(function(encrypted) {
            if (!encrypted || encrypted.trim() === '') return;

            var decrypted = self.dec(encrypted.trim());
            if (!decrypted) return;

            var data;
            try {
                data = JSON.parse(decrypted);
            } catch (e) {
                console.error('Moustique: Failed to parse response:', e);
                return;
            }

            // Process messages per topic
            for (var topicKey in data) {
                if (!data.hasOwnProperty(topicKey)) continue;
                var messages = data[topicKey];

                for (var i = 0; i < messages.length; i++) {
                    var msg = messages[i];
                    var msgTopic = msg.topic || topicKey;
                    var msgContent = msg.message || '';
                    var msgFrom = msg.from || '';

                    // Find matching callbacks
                    self.callbacks.forEach(function(callbacks, subscribedTopic) {
                        if (self.topicMatches(subscribedTopic, msgTopic)) {
                            callbacks.forEach(function(cb) {
                                try {
                                    cb(msgTopic, msgContent, msgFrom);
                                } catch (e) {
                                    console.error('Moustique: Callback error:', e);
                                }
                            });
                        }
                    });
                }
            }
        }).catch(function(err) {
            if (self.debug) console.error('Moustique: Pickup error:', err);
        });
    };

    // Publish a message
    Moustique.prototype.publish = function(topic, message) {
        var self = this;

        var payload = new URLSearchParams({
            topic: this.enc(topic),
            message: this.enc(message),
            from: this.enc(this.clientName),
            updated_time: this.enc(Math.floor(Date.now() / 1000).toString()),
            updated_nicedatetime: this.enc(this.getNiceDateTime())
        });

        if (this.username && this.password) {
            payload.append('username', this.enc(this.username));
            payload.append('password', this.enc(this.password));
        }

        return fetch(this.baseUrl + '/POST', {
            method: 'POST',
            body: payload,
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
        }).then(function(res) {
            if (res.ok) {
                if (self.debug) console.log('Moustique: Published to ' + topic);
                return true;
            } else {
                console.error('Moustique: Publish failed:', res.status);
                return false;
            }
        }).catch(function(err) {
            console.error('Moustique: Publish error:', err);
            return false;
        });
    };

    // MQTT-style topic matching (+ = single level, # = multi-level)
    Moustique.prototype.topicMatches = function(pattern, topic) {
        var patternParts = pattern.split('/');
        var topicParts = topic.split('/');

        for (var i = 0; i < patternParts.length; i++) {
            if (patternParts[i] === '#') return true;
            if (i >= topicParts.length) return false;
            if (patternParts[i] === '+') continue;
            if (patternParts[i] !== topicParts[i]) return false;
        }
        return patternParts.length === topicParts.length;
    };

    // Get client name
    Moustique.prototype.getClientName = function() {
        return this.clientName;
    };

    // Expose to global scope
    global.Moustique = Moustique;

})(typeof window !== 'undefined' ? window : this);
