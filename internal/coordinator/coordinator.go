package coordinator

import (
	"log"
	"strings"
	"time"

	"rn-lb/internal/config"
	"rn-lb/internal/health"
)

// Run käynnistää koordinaattorin, joka valvoo servereiden health checkejä
func Run(cfg *config.Config, stop <-chan struct{}) {
	log.Println("Coordinator starting...")

	for {
		select {
		case <-stop:
			log.Println("Coordinator stopping...")
			return
		default:
			checkAll(cfg)
			time.Sleep(cfg.Interval)
		}
	}
}

func checkAll(cfg *config.Config) {
	for _, e := range cfg.Entities {
		for _, server := range e.Servers {
			addr := server.Address

			// Jos osoitteessa ei ole protokollaa, lisätään se
			if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
				addr = "http://" + addr
			}

			// Suoritetaan health check
			alive := health.Check(addr, server.Check, e.Health)

			if alive {
				log.Printf("[OK]   %s (%s)", addr, server.Name)
			} else {
				log.Printf("[FAIL] %s (%s)", addr, server.Name)
			}
		}
	}
}
