# Notifier

A D-Bus notification server that broadcasts filtered notifications to WebSocket clients. Supports pattern-based subscriptions for URLs and notification summaries.

## Architecture

The program uses multiple goroutines and a subscription-based model:

1. **Main Goroutine**
   - Handles D-Bus notification server setup
   - Listens for system signals (Ctrl+C)
   - Blocks until shutdown signal received

2. **WebSocket Server**
   - Accepts client connections
   - Handles subscription/unsubscription requests
   - Each client can subscribe to multiple notification patterns

3. **Client Management**
   - Thread-safe client tracking
   - Pattern-based notification filtering
   - Full notification data forwarding

## WebSocket Protocol

### Subscribe to Notifications
```json
{
  "type": "subscribe",
  "payload": {
    "url_pattern": "x.com",
    "summary_pattern": "optional pattern"  // Optional
  }
}
```

### Unsubscribe from Notifications
```json
{
  "type": "unsubscribe",
  "payload": {
    "url_pattern": "x.com"
  }
}
```

### Notification Format
Clients receive complete notification data:
```json
{
  "app_name": "Firefox",
  "id": 123,
  "icon": "icon-path",
  "summary": "New Message",
  "body": "https://x.com/...",
  "actions": ["default"],
  "hints": {"urgency": 1},
  "expire_timeout": 5000
}
```

## Features
- **Pattern-Based Filtering**: Subscribe to notifications containing specific URLs
- **Optional Summary Filtering**: Further filter by notification summary text
- **Complete Notification Data**: Receive all notification fields, not just the body
- **Multiple Subscriptions**: Each client can have multiple active subscriptions
- **Real-time Updates**: Instant notification delivery to subscribed clients
- **Thread-Safe**: Concurrent client and subscription management

## Data Flow
1. D-Bus notification arrives → `Notify` method called
2. Notification converted to JSON format
3. Pattern matching against client subscriptions
4. Filtered notifications sent to matching clients

## Usage Example
```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/ws');

// Subscribe to Twitter notifications
ws.send(JSON.stringify({
  type: 'subscribe',
  payload: {
    url_pattern: 'x.com',
    summary_pattern: 'New Tweet'  // Optional
  }
}));

// Handle incoming notifications
ws.onmessage = (event) => {
  const notification = JSON.parse(event.data);
  console.log('Received notification:', notification);
};
