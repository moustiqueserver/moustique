# Moustique JavaScript Client

![License](https://img.shields.io/badge/license-MIT-green)
![Node](https://img.shields.io/badge/node-%3E%3D18-blue)
![Browser](https://img.shields.io/badge/browser-compatible-brightgreen)
![Zero Dependencies](https://img.shields.io/badge/dependencies-zero-success)

A lightweight, zero-dependency HTTP-based **publish/subscribe client** for the  
[Moustique server](https://github.com/yourusername/moustique).

Designed to work seamlessly in both **Node.js (v18+)** and **modern browsers** using native `fetch`.

---

## ✨ Features

- Full **publish / subscribe** functionality
- Persistent key-value storage via `putval` / `get_val`
- Automatic **resubscribe** on reconnect-like events
- Zero external dependencies
- Promise-based **async / await** API
- Browser-safe (no Node-only APIs)
- TypeScript-friendly (interface-compatible)

---

## 📦 Installation

This client lives inside the **Moustique monorepo**.

```bash
git clone https://github.com/yourusername/moustique.git
cd moustique/clients/javascript
```

### Browser (direct import)

```html
<script type="module">
  import { Moustique } from 'https://raw.githubusercontent.com/yourusername/moustique/main/clients/javascript/moustique/index.js';
</script>
```

> Replace `yourusername` with the actual repository owner.

---

## 🚀 Usage

### Node.js

```js
import { Moustique } from './moustique/index.js';

const client = new Moustique({
  ip: '192.168.1.79',
  port: '33334',
  clientName: 'MyNodeApp'
});

client.subscribe('/test/topic', (topic, message, from) => {
  console.log(`Received on ${topic}: ${message} (from ${from})`);
});

await client.publish('/test/topic', 'Hello from Node.js!');
await client.putval('/settings/theme', 'dark');

// Poll for new messages
setInterval(() => client.pickup(), 1000);
```

---

### Browser

Live example: `examples/browser.html`

```html
<script type="module">
  import { Moustique } from '../moustique/index.js';

  const client = new Moustique({ clientName: 'BrowserUser' });

  client.subscribe('/chat', (topic, message, from) => {
    console.log(`${from}: ${message}`);
  });

  await client.publish('/chat', 'Hi from the browser!');
</script>
```

---

## 📚 API Documentation

See the full API reference in **[API.md](API.md)**

---

## 🧪 Running Tests

```bash
cd clients/javascript
node tests/test_client.js
```

---

## 🌍 Compatibility

- **Node.js**: v18.0.0 or newer (native `fetch` required)
- **Browsers**: Chrome, Firefox, Safari, Edge (latest versions)

---

## 📄 License

MIT License – see the `LICENSE` file in the repository root.
