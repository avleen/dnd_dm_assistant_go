package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dnd_dm_assistant_go/internal/bot"
	"dnd_dm_assistant_go/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize bot
	dndBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start bot
	if err := dndBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	// Wait for interrupt signal
	fmt.Println("D&D DM Assistant Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanup
	fmt.Println("Shutting down...")
	dndBot.Stop()
}
