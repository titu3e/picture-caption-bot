package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "", "Path to config")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("Path to config must be specified")
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Token == "Your token here" || cfg.Token == "" {
		log.Fatal("You are using default config, please copy it and fill with appropriate values")
	}

	bot, err := New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done, err := bot.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	err = nil

	select {
	case s := <-sig:
		log.Printf("Recieved %v signal, stopping bot with 10 seconds timeout", s)
		cancel()
		select {
		case err = <-done:
		case <-time.After(10 * time.Second):
			log.Fatalf("Bot did not stop after 10 seconds, halting")
		}
	case err = <-done:
	}

	if err != nil {
		log.Printf("Bot stopped with error: %v", err)
	} else {
		log.Print("Bot stopped without error")
	}
}
