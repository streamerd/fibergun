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

| Property          | Type     | Description                                                                  | Default    |
|------------------|----------|------------------------------------------------------------------------------|------------|
| Next             | `func(*fiber.Ctx) bool` | A function to skip this middleware when returned true         | `nil`      |
| WebSocketEndpoint| `string` | The endpoint where GunDB websocket connections will be handled              | `"/gun"`   |
| StaticPath       | `string` | The path to serve the GunDB client files                                   | `"./public"` |

## Example

```go
package main

import (
    "log"
    
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/contrib/fibergun"
)

func main() {
    app := fiber.New()

    // Initialize GunDB middleware
    app.Use(fibergun.New(&fibergun.Config{
        StaticPath: "./public",
    }))

    // Serve static files
    app.Static("/", "./public")

    log.Fatal(app.Listen(":3000"))
}
```

### Client Example

```html
<!DOCTYPE html>
<html>
<head>
    <title>GunDB + Fiber Example</title>
    <script src="https://cdn.jsdelivr.net/npm/gun/gun.js"></script>
</head>
<body>
    <div id="app">
        <textarea id="notepad"></textarea>
    </div>

    <script>
        const gun = GUN({
            peers: [`ws://${window.location.host}/gun`]
        });

        const notepad = document.getElementById('notepad');
        const notes = gun.get('notes');

        // Update GunDB when textarea changes
        notepad.addEventListener('input', (e) => {
            notes.put({
                text: e.target.value
            });
        });

        // Update textarea when GunDB data changes
        notes.on(data => {
            if (data.text !== undefined && notepad.value !== data.text) {
                notepad.value = data.text;
            }
        });
    </script>
</body>
</html>
```

## WebSocket Handler (internal)

The middleware sets up a WebSocket handler that:
1. Manages peer connections
2. Routes messages between peers
3. Handles connection/disconnection events
4. Facilitates data synchronization

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