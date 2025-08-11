package main

import (
	"go4pack/pkg/app"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create a channel to listen for OS signals
	sigs := make(chan os.Signal, 1)
	// Notify the channel when an interrupt or terminate signal is received
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Run the app in a goroutine
	go app.RunAPI()

	// Block until a signal is received
	<-sigs

	// Optionally, you can add cleanup code here
}
