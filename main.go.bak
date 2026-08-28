package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrabhi2k3/telegofer/client"
	"github.com/mrabhi2k3/telegofer/mtproto/session"
)

func main() {
	// Configuration - Replace with your credentials or set environment variables
	apiID := 20389440
	apiHash := "a1a06a18eb9153e9dbd447cfd5da2457"
	botToken := "8489438859:AAHxM60SX_Ublz3hWxtSvUTD9jsk_A2SKRM"

	fmt.Println("🤖 Starting TeleGofer Sample Bot...")
	fmt.Printf("API ID: %d | Bot Token: %s...\n", apiID, botToken[:10])

	// Initialize TeleGofer Client Configuration
	cfg := client.Config{
		APIID:          apiID,
		APIHash:        apiHash,
		Session:        session.NewFile("bot_session.dat"), // Save session state locally
		ConnectTimeout: 15 * time.Second,
		RequestTimeout: 30 * time.Second,
	}

	// Create Client
	bot, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create bot client: %v", err)
	}

	// Set up context cancellation for graceful shutdown (Ctrl+C)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Connect to Telegram MTProto Server
	log.Println("🔌 Connecting to Telegram servers...")
	if err := bot.Start(ctx); err != nil {
		log.Fatalf("❌ Connection failed: %v", err)
	}
	defer bot.Close()

	log.Println("✅ Bot connected successfully!")
	log.Println("💡 Ready to take commands. Press Ctrl+C to stop.")

	// Keep the bot process running
	<-ctx.Done()
	log.Println("👋 Shutting down bot gracefully...")
}
