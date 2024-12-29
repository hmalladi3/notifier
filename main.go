package main

import (
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

type NotificationServer struct {
	conn     *dbus.Conn
	clients  map[*websocket.Conn]bool
	mutex    sync.Mutex
	broadcast chan string
}

func NewNotificationServer(conn *dbus.Conn) *NotificationServer {
	return &NotificationServer{
		conn:      conn,
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan string),
	}
}

// Implementation of the Notify method that dunst calls
func (n *NotificationServer) Notify(appName string, replacesID uint32, icon string, summary string, body string, actions []string, hints map[string]dbus.Variant, expireTimeout int32) (uint32, *dbus.Error) {
	// Check if this is a Discord notification
	if strings.Contains(body, "discord.com") {
		// Extract the actual message text
		parts := strings.Split(body, "\n\n")
		if len(parts) >= 2 {
			messageText := parts[len(parts)-1]
			// Send to broadcast channel
			n.broadcast <- messageText
		}
	} else {
		fmt.Printf("\nNEW NOTIFICATION\n")
		fmt.Printf("App: %s\n", appName)
		fmt.Printf("Summary: %s\n", summary)
		fmt.Printf("Body: %s\n", body)
		fmt.Printf("Icon: %s\n", icon)
		if len(hints) > 0 {
			fmt.Printf("Hints:\n")
			for k, v := range hints {
				fmt.Printf("  %s: %v\n", k, v.Value())
			}
		}
	}
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
	n.clients[conn] = true
	n.mutex.Unlock()

	// Remove client when connection closes
	defer func() {
		n.mutex.Lock()
		delete(n.clients, conn)
		n.mutex.Unlock()
	}()

	// Keep connection alive
	for {
		// Read messages from client (if any)
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (n *NotificationServer) broadcastMessages() {
	for message := range n.broadcast {
		n.mutex.Lock()
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
