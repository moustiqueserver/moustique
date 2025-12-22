# Moustique JavaScript Client – API

## Constructor

```js
new Moustique(options)
```

### Options

| Option       | Type   | Required | Description |
|--------------|--------|----------|-------------|
| `ip`         | string | No       | Server IP address |
| `port`       | string | No       | Server port |
| `clientName` | string | No       | Client identifier (auto-generated if omitted) |

---

## Methods

### publish

```js
publish(topic: string, message: string): Promise<void>
```

Publishes a message to a topic.

---

### subscribe

```js
subscribe(
  topic: string,
  callback: (topic: string, message: string, from: string) => void
): Promise<void>
```

Subscribes to a topic and registers a callback for incoming messages.

---

### putval

```js
putval(topic: string, value: string): Promise<void>
```

Stores a persistent key-value pair on the server.

---

### pickup

```js
pickup(): Promise<void>
```

Fetches and processes pending messages.  
Typically called on a timer.

---

### resubscribe

```js
resubscribe(): Promise<void>
```

Re-subscribes to all previously subscribed topics.  
Useful after reconnects or network interruptions.

---

### getClientName

```js
getClientName(): string
```

Returns the current client identifier.

---

## Notes

- All network methods are asynchronous
- Designed to be polling-friendly
- Safe to use in browser environments
