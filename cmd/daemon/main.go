package main

import (
	"log"
	"rn-lb/internal/config"
	"rn-lb/internal/coordinator"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	cfg.ApplyDefaults()

	coordinator.Run(cfg)
}
