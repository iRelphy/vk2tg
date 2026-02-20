package main

// Entry point of the program.
//
// We keep main.go very small:
// - load env
// - read config
// - create app
// - handle OS signals (Ctrl+C) and shut down cleanly

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iRelphy/vk2tg/internal/app"
	"github.com/iRelphy/vk2tg/internal/config"
)

func main() {
	// Optional: load main.env from current folder.
	config.LoadDotEnv()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	a, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Context is a standard Go way to stop background loops.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen to OS signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("Stopping...")
		cancel()
	}()

	// Run blocks until VK LongPoll stops.
	if err := a.Run(ctx); err != nil {
		log.Fatalf("run: %v", err)
	}
}
