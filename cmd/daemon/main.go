package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"rn-lb/internal/config"
	"rn-lb/internal/coordinator"
	"rn-lb/internal/provider"
)

func main() {
	// 1. Load config
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Init provider
	p, err := provider.NewProvider(cfg.Global.Provider)
	if err != nil {
		log.Fatalf("failed to init provider: %v", err)
	}
	log.Printf("Provider initialized: %T", p)

	// 3. Start coordinator
	stop := make(chan struct{})
	daemonID := cfg.Daemon.ID // later can load/generate file

	go coordinator.Run(cfg, p, daemonID, stop)

	// 4. Signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	log.Println("Shutting down daemon...")
	close(stop)
}
