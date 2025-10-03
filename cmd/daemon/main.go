package main

import (
	"log"

	"rn-lb/internal/config"
	"rn-lb/internal/coordinator"
	//	"rn-lb/internal/provider/cloudflare"
)

func main() {
	// Ladataan config.yaml
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Käynnistetään koordinointi
	var secondArg /* TODO: replace with the correct type and value */
	coordinator.Run(cfg, secondArg)
}
