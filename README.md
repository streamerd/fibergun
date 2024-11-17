# Fiber GunDB Middleware

[![Go Reference](https://pkg.go.dev/badge/github.com/gofiber/contrib/fibergun.svg)](https://pkg.go.dev/github.com/gofiber/contrib/fibergun)
[![Go Report Card](https://goreportcard.com/badge/github.com/gofiber/contrib/fibergun)](https://goreportcard.com/report/github.com/gofiber/contrib/fibergun)

GunDB middleware for [Fiber](https://github.com/gofiber/fiber) web framework. This middleware enables easy integration of [GunDB](https://gun.eco/), a decentralized database, with your Fiber applications.

## Install

This middleware supports Fiber v2.

```bash
go get -u github.com/gofiber/fiber/v2
go get -u github.com/gofiber/contrib/fibergun
```

## Signature

```go
fibergun.New(config ...*fibergun.Config) fiber.Handler
```

## Config

| Property | Type | Description | Default |
|------------------|----------|------------------------------------------------------------------------------|------------|
| Next | `func(*fiber.Ctx) bool` | A function to skip this middleware when returned true | `nil` |
| WebSocketEndpoint | `string` | The endpoint where GunDB websocket connections will be handled | `"/gun"` |
| StaticPath | `string` | The path to serve the GunDB client files | `"./public"` |
| HeartbeatInterval | `time.Duration` | Interval for sending heartbeat pings | `15 * time.Second` |
| PeerTimeout | `time.Duration` | Duration after which a peer is considered inactive | `60 * time.Second` |
| MaxMessageSize | `int64` | Maximum size of a WebSocket message in bytes | `1024 * 1024` (1MB) |
| EnableCompression | `bool` | Enables WebSocket compression | `true` |
| BufferSize | `int` | Sets the read/write buffer size for WebSocket connections | `1024 * 16` (16KB) |
| Debug | `bool` | Enables detailed logging | `false` |
| ReconnectAttempts | `int` | Number of times to attempt reconnection | `5` |
| ReconnectInterval | `time.Duration` | Time to wait between reconnection attempts | `2 * time.Second` |
| DataReplication | `DataReplicationConfig` | Configures how data is replicated between peers | See below |

### DataReplicationConfig

| Property | Type | Description | Default |
|----------|------|-------------|----------|
| Enabled | `bool` | Determines if data should be replicated between peers | `true` |
| SyncInterval | `time.Duration` | How often to sync data between peers | `30 * time.Second` |
| MaxRetries | `int` | Maximum number of sync retries | `3` |
| BatchSize | `int` | Maximum number of items to sync at once | `100` |

## Example

### Server Setup

```go
package main

import (
    "log"
    "time"
    
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/contrib/fibergun"
)

func main() {
    app := fiber.New(fiber.Config{
        DisableStartupMessage: false,
    })

    // Add middleware
    app.Use(logger.New(logger.Config{
        Format:     "${time} ${status} - ${latency} ${method} ${path}\n",
        TimeFormat: "15:04:05",
        TimeZone:   "Local",
    }))

    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
        AllowHeaders: "*",
    }))

    // Initialize GunDB middleware
    app.Use(fibergun.New(&fibergun.Config{
        StaticPath:        "./public",
        WebSocketEndpoint: "/gun",
        HeartbeatInterval: 15 * time.Second,
        PeerTimeout:       60 * time.Second,
        EnableCompression: true,
        BufferSize:        1024 * 16,
        Debug:            true,
        DataReplication: fibergun.DataReplicationConfig{
            Enabled:      true,
            SyncInterval: 5 * time.Second,
            MaxRetries:   5,
            BatchSize:    100,
        },
    }))

    app.Static("/", "./public")

    log.Printf("Starting server on :3000...")
    log.Fatal(app.Listen(":3000"))
}
```

### Client Example

Create `public/index.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <title>GunDB + Fiber Example</title>
    <script src="https://cdn.jsdelivr.net/npm/gun/gun.js"></script>
</head>
<body>
    <div class="container">
        <h1>GunDB + Fiber Example</h1>
        <div id="status"></div>

        <div class="message-form">
            <input type="text" id="nameInput" placeholder="Your name" />
            <textarea id="messageInput" placeholder="Type your message"></textarea>
            <button onclick="sendMessage()">Send Message</button>
        </div>

        <div id="messages"></div>
    </div>

    <script>
        const gun = GUN({
            peers: [`ws://${window.location.host}/gun`],
            localStorage: true,
            debug: true,
            axe: false,
            retry: 1000
        });

        const messages = gun.get('chat');

        function sendMessage() {
            const name = nameInput.value.trim() || 'Anonymous';
            const text = messageInput.value.trim();
            
            if (!text) return;

            const messageId = Date.now().toString();
            messages.get(messageId).put({
                name: name,
                text: text,
                timestamp: Date.now(),
                id: messageId
            });

            messageInput.value = '';
        }

        messages.map().on((data, key) => {
            if (!data || !data.timestamp) return;
            displayMessage(data);
        });
    </script>
</body>
</html>
```

## Features

1. Real-time peer-to-peer communication
2. Message persistence across peers
3. Automatic peer discovery and sync
4. WebSocket compression support
5. Configurable heartbeat and timeouts
6. Data replication with retry mechanisms
7. Debug mode for development
8. CORS support
9. Connection state management
10. Automatic reconnection handling

## WebSocket Handler

The middleware provides a WebSocket handler that:
1. Manages peer connections and lifecycle
2. Routes messages between peers
3. Handles connection/disconnection events
4. Facilitates data synchronization
5. Maintains heartbeat for connection health
6. Handles message retries and acknowledgments

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -am 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Acknowledgments

- [Fiber Web Framework](https://github.com/gofiber/fiber)
- [GunDB](https://gun.eco/)