# Notifier

listens to dbus notifications from dunst and broadcasts them to websocket clients

## Low Latency Architecture

The program uses multiple goroutines to achieve low latency:

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

### Data Flow
1. D-Bus notification arrives → `Notify` method called
2. Message extracted and sent to broadcast channel (non-blocking)
3. Broadcast goroutine picks up message from channel
4. Message distributed to all WebSocket clients in parallel

This architecture ensures:
- Notifications are processed immediately
- Broadcasting doesn't block new notifications
- Client connections don't affect notification processing
- System remains responsive under load
