package main_test

import (
	"fmt"
	"log"

	"github.com/hmalladi3/notifier"
)

func ExampleNotificationServer() {
	// Create a new notification server
	server, err := notifier.NewNotificationServer()
	if err != nil {
		log.Fatal("Failed to create notification server:", err)
	}
	defer server.Stop()

	// Register a handler for notifications
	server.OnNotification(func(message string) {
		fmt.Printf("Received notification: %s\n", message)
	})

	// Start the notification server
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start notification server:", err)
	}

	fmt.Println("Notification server is running")

	// Send a test notification (you would typically do this from another client)
	fmt.Println("Server is ready to handle notifications")

	// Output:
	// Notification server is running
	// Server is ready to handle notifications
}
