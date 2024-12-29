package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/godbus/dbus/v5"
)

const (
	dbusInterface = "org.freedesktop.Notifications"
	dbusPath      = "/org/freedesktop/Notifications"
)

type NotificationServer struct {
	conn *dbus.Conn
}

// Implementation of the Notify method that dunst calls
func (n *NotificationServer) Notify(appName string, replacesID uint32, icon string, summary string, body string, actions []string, hints map[string]dbus.Variant, expireTimeout int32) (uint32, *dbus.Error) {
	// Check if this is a Discord notification
	if strings.Contains(body, "discord.com") {
		// Extract the actual message text
		// The format is: <a href="https://discord.com/">discord.com</a>\n\nACTUAL_MESSAGE
		parts := strings.Split(body, "\n\n")
		if len(parts) >= 2 {
			messageText := parts[len(parts)-1]
			fmt.Printf("Discord: %s\n", messageText)
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

func main() {
	// Connect to the session bus
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Fatal("Failed to connect to session bus:", err)
	}
	defer conn.Close()

	// Create our notification server
	server := &NotificationServer{conn: conn}

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

	fmt.Println("Notification listener running... (Press Ctrl+C to exit)")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down notification listener...")
}
