# Notifier

A Go library for handling system notifications with multiple implementation options.

## Implementations

### 1. Direct Notification (Main Version)
The main implementation provides a simple, direct notification handling system:
- Direct callback-based notification handling
- No external dependencies beyond standard Go libraries
- Perfect for applications that need to process notifications directly
- Simple integration with minimal setup

Example usage:
```go
server, err := notifier.NewNotificationServer()
if err != nil {
    log.Fatal(err)
}
defer server.Stop()

server.OnNotification(func(message string) {
    fmt.Printf("Received notification: %s\n", message)
})

if err := server.Start(); err != nil {
    log.Fatal(err)
}
```

### 2. WebSocket Version (Alternative Implementation)
The WebSocket implementation (`websocket-version` branch) provides network-based notification distribution:
- Listens to dbus notifications from dunst
- Broadcasts notifications to WebSocket clients
- Suitable for distributed systems and web applications
- Real-time notification delivery over network

## Architecture

### Direct Notification Version
1. **Notification Server**
   - Handles D-Bus notification setup
   - Manages notification callbacks
   - Direct processing without network overhead

2. **Callback System**
   - Register handlers for notifications
   - Immediate processing of notifications
   - No intermediate message queuing

### WebSocket Version
1. **WebSocket Server**
   - Handles client connections
   - Broadcasts notifications to connected clients
   - Network-based distribution

2. **Broadcast System**
   - Channel-based message broadcasting
   - Parallel client message distribution
   - Non-blocking notification processing

The WebSocket version uses multiple goroutines to achieve low latency:

1. **Main Goroutine**
   - Handles D-Bus notification server setup
   - Listens for system signals (Ctrl+C)
   - Blocks until shutdown signal received

2. **WebSocket Server Goroutine**
   ```go
   go func() {
       log.Printf("Starting WebSocket server on %s", wsPort)
       if err := http.ListenAndServe(wsPort, nil); err != nil {
           log.Fatal("WebSocket server failed:", err)
       }
   }()
   ```
   - Runs HTTP server for WebSocket connections
   - Accepts new client connections without blocking main thread
   - Each client connection gets its own goroutine for handling messages

3. **Broadcast Goroutine**
   ```go
   go server.broadcastMessages()
   ```
   - Dedicated goroutine for message broadcasting
   - Listens to broadcast channel continuously
   - Distributes messages to all connected clients
   - Channel-based communication prevents blocking between notification receipt and broadcasting

This architecture ensures:
- Notifications are processed immediately
- Broadcasting doesn't block new notifications
- Client connections don't affect notification processing
- System remains responsive under load

## Getting Started

Check the `examples` directory for implementation examples of both versions.

To use the WebSocket version, switch to the `websocket-version` branch:
```bash
git checkout websocket-version
