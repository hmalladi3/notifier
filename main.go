package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/godbus/dbus/v5"
	"github.com/gorilla/websocket"
)

const (
	dbusInterface = "org.freedesktop.Notifications"
	dbusPath      = "/org/freedesktop/Notifications"
	wsPort        = ":8080"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for testing
	},
}

type Subscription struct {
	URLPattern     string `json:"url_pattern"`
	SummaryPattern string `json:"summary_pattern,omitempty"`
}

type Client struct {
	conn          *websocket.Conn
	subscriptions []Subscription
}

type NotificationData struct {
	AppName       string                 `json:"app_name"`
	ID           uint32                 `json:"id"`
	Icon         string                 `json:"icon"`
	Summary      string                 `json:"summary"`
	Body         string                 `json:"body"`
	Actions      []string               `json:"actions"`
	Hints        map[string]interface{} `json:"hints"`
	ExpireTimeout int32                 `json:"expire_timeout"`
}

type NotificationServer struct {
	conn      *dbus.Conn
	clients   map[*websocket.Conn]*Client
	mutex     sync.Mutex
	broadcast chan string
}

type WSMessage struct {
	Type    string      `json:"type"`    // "subscribe" or "unsubscribe"
	Payload Subscription `json:"payload"`
}

func NewNotificationServer(conn *dbus.Conn) *NotificationServer {
	return &NotificationServer{
		conn:      conn,
		clients:   make(map[*websocket.Conn]*Client),
		broadcast: make(chan string),
	}
}

// Implementation of the Notify method that dunst calls
func (n *NotificationServer) Notify(appName string, replacesID uint32, icon string, summary string, body string, actions []string, hints map[string]dbus.Variant, expireTimeout int32) (uint32, *dbus.Error) {
	// Debug: Print all notification details
	fmt.Printf("\n[DBUS NOTIFICATION DETAILS]\n"+
		"Application: %s\n"+
		"ID: %d\n"+
		"Icon: %s\n"+
		"Summary: %s\n"+
		"Body: %s\n"+
		"Actions: %v\n"+
		"Hints: %v\n"+
		"Timeout: %d\n",
		appName, replacesID, icon, summary, body, actions, hints, expireTimeout)

	// Convert hints to a map[string]interface{}
	hintsMap := make(map[string]interface{})
	for k, v := range hints {
		hintsMap[k] = v.Value()
	}

	// Create notification data
	notif := NotificationData{
		AppName:       appName,
		ID:           replacesID,
		Icon:         icon,
		Summary:      summary,
		Body:         body,
		Actions:      actions,
		Hints:        hintsMap,
		ExpireTimeout: expireTimeout,
	}

	// Marshal notification to JSON
	notifJSON, err := json.Marshal(notif)
	if err != nil {
		log.Printf("Failed to marshal notification: %v", err)
		return 1, nil
	}

	// Send to subscribed clients
	n.mutex.Lock()
	for _, client := range n.clients {
		// Check each subscription
		for _, sub := range client.subscriptions {
			if strings.Contains(body, sub.URLPattern) {
				// If summary pattern is specified, check it
				if sub.SummaryPattern != "" {
					if !strings.Contains(summary, sub.SummaryPattern) {
						continue
					}
				}
				// Send notification
				err := client.conn.WriteMessage(websocket.TextMessage, notifJSON)
				if err != nil {
					log.Printf("Failed to send message to client: %v", err)
					client.conn.Close()
					delete(n.clients, client.conn)
				}
				break // Don't send multiple times to same client
			}
		}
	}
	n.mutex.Unlock()

	return 1, nil
}

func (n *NotificationServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register new client
	n.mutex.Lock()
	n.clients[conn] = &Client{conn: conn}
	clientCount := len(n.clients)
	n.mutex.Unlock()
	
	fmt.Printf("New WebSocket client connected (total clients: %d)\n", clientCount)

	// Remove client when connection closes
	defer func() {
		n.mutex.Lock()
		delete(n.clients, conn)
		clientCount := len(n.clients)
		n.mutex.Unlock()
		fmt.Printf("WebSocket client disconnected (remaining clients: %d)\n", clientCount)
	}()

	// Keep connection alive
	for {
		// Read messages from client (if any)
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var wsMessage WSMessage
		err = json.Unmarshal(message, &wsMessage)
		if err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch wsMessage.Type {
		case "subscribe":
			n.mutex.Lock()
			client := n.clients[conn]
			client.subscriptions = append(client.subscriptions, wsMessage.Payload)
			n.mutex.Unlock()
			log.Printf("Client subscribed to %s\n", wsMessage.Payload.URLPattern)
		case "unsubscribe":
			n.mutex.Lock()
			client := n.clients[conn]
			for i, subscription := range client.subscriptions {
				if subscription.URLPattern == wsMessage.Payload.URLPattern {
					client.subscriptions = append(client.subscriptions[:i], client.subscriptions[i+1:]...)
					break
				}
			}
			n.mutex.Unlock()
			log.Printf("Client unsubscribed from %s\n", wsMessage.Payload.URLPattern)
		default:
			log.Printf("Unknown message type: %s\n", wsMessage.Type)
		}
	}
}

func (n *NotificationServer) broadcastMessages() {
	for message := range n.broadcast {
		n.mutex.Lock()
		clientCount := len(n.clients)
		if clientCount > 0 {
			fmt.Printf("Broadcasting to %d clients...\n", clientCount)
		}
		for client := range n.clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Printf("Failed to send message to client: %v", err)
				client.Close()
				delete(n.clients, client)
			}
		}
		n.mutex.Unlock()
	}
}

func main() {
	// Connect to the session bus
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatal("Failed to connect to session bus:", err)
	}
	defer conn.Close()

	// Create our notification server
	server := NewNotificationServer(conn)

	// Export the notification server on the bus
	err = conn.Export(server, dbus.ObjectPath(dbusPath), dbusInterface)
	if err != nil {
		log.Fatal("Failed to export server:", err)
	}

	// Request the notification service name
	reply, err := conn.RequestName(dbusInterface,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Fatal("Failed to request name:", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Name already taken. Is another notification daemon running?")
	}

	// Start WebSocket server
	http.HandleFunc("/ws", server.handleWebSocket)
	go func() {
		log.Printf("Starting WebSocket server on %s", wsPort)
		if err := http.ListenAndServe(wsPort, nil); err != nil {
			log.Fatal("WebSocket server failed:", err)
		}
	}()

	// Start broadcasting messages
	go server.broadcastMessages()

	fmt.Printf("Notification listener running...\n")
	fmt.Printf("WebSocket server running on ws://localhost%s/ws\n", wsPort)
	fmt.Printf("Press Ctrl+C to exit\n")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	close(server.broadcast)
}
