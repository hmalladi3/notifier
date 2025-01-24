package notifier

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	dbusInterface = "org.freedesktop.Notifications"
	dbusPath      = "/org/freedesktop/Notifications"
)

// NotificationHandler is a callback function type for handling notifications
type NotificationHandler func(message string)

type NotificationServer struct {
	conn     *dbus.Conn
	handlers []NotificationHandler
}

// NewNotificationServer creates a new notification server instance
func NewNotificationServer() (*NotificationServer, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %v", err)
	}

	server := &NotificationServer{
		conn:     conn,
		handlers: make([]NotificationHandler, 0),
	}

	return server, nil
}

// OnNotification registers a handler function that will be called for each notification
func (n *NotificationServer) OnNotification(handler NotificationHandler) {
	n.handlers = append(n.handlers, handler)
}

// Start initializes the notification server and starts listening for notifications
func (n *NotificationServer) Start() error {
	if err := n.conn.Export(n, dbusPath, dbusInterface); err != nil {
		return fmt.Errorf("failed to export notification server: %v", err)
	}

	reply, err := n.conn.RequestName(dbusInterface, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request name: %v", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("name already taken")
	}

	return nil
}

// Stop gracefully shuts down the notification server
func (n *NotificationServer) Stop() error {
	if err := n.conn.Close(); err != nil {
		return fmt.Errorf("failed to close dbus connection: %v", err)
	}
	return nil
}

// Implementation of the Notify method that dunst calls
func (n *NotificationServer) Notify(appName string, replacesID uint32, icon string, summary string, body string, actions []string, hints map[string]dbus.Variant, expireTimeout int32) (uint32, *dbus.Error) {
	// Check if this is a Twitter/X notification
	if strings.Contains(body, "x.com") {
		// Print to terminal for debugging
		fmt.Printf("\n[DEBUG] Raw notification body:\n%s\n", body)

		// Find the content after x.com link
		parts := strings.Split(body, "</a>\n\n")
		if len(parts) >= 2 {
			messageText := strings.TrimSpace(parts[1])
			// Print to terminal
			fmt.Printf("[TWITTER] %s\n", messageText)

			// Call all registered handlers
			for _, handler := range n.handlers {
				handler(messageText)
			}
		}
	} else {
		// Debug: print other notifications
		fmt.Printf("Other notification from %s: %s\n", appName, body)
	}
	return 1, nil
}
