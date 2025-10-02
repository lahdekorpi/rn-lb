package coordinator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yourusername/rn-lb/internal/config"
	"github.com/yourusername/rn-lb/internal/health"
	"github.com/yourusername/rn-lb/internal/provider"
)

// Run starts the main coordination loop for all entities.
func Run(cfg *config.Config, cfProvider provider.Provider) {
	rateLimiter := time.Tick(500 * time.Millisecond)

	for {
		ctx := context.Background()
		fmt.Println("\nNew cycle:", time.Now().Format("15:04:05"))

		for _, e := range cfg.Entities {
			zoneID := e.Provider.Cloudflare.ZoneID
			fmt.Printf("\nEntity: %s (zone: %s)\n", e.Name, zoneID)

			// Determine health check parameters
			timeout := e.Health.Timeout
			if timeout == 0 {
				timeout = cfg.Global.Health.Timeout
			}
			retries := e.Health.Retries
			if retries == 0 {
				retries = cfg.Global.Health.Retries
			}
			retryWait := e.Health.RetryWait
			if retryWait == 0 {
				retryWait = cfg.Global.Health.RetryWait
			}

			healthyServers := []string{}
			for _, server := range e.Servers {
				url := server.Address
				if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
					url = "http://" + url
				}

				if health.CheckServer(url, timeout, retries, retryWait) {
					fmt.Printf("Server %s responded OK\n", server.Address)
					healthyServers = append(healthyServers, server.Address)
				} else {
					fmt.Printf("Server %s did not respond\n", server.Address)
				}
			}

			// Sync DNS records
			if len(healthyServers) == 0 {
				fmt.Println("No healthy servers -> clearing A records and marking TXT as DOWN")
				if err := cfProvider.SyncARecords(ctx, zoneID, e.Hostname, []string{}, rateLimiter); err != nil {
					log.Printf("Failed to clear A records: %v", err)
				}
				txtHost := "health." + e.Hostname
				if err := cfProvider.SyncTXTRecord(ctx, zoneID, txtHost, "DOWN", rateLimiter); err != nil {
					log.Printf("Failed to update TXT record: %v", err)
				}
			} else {
				if err := cfProvider.SyncARecords(ctx, zoneID, e.Hostname, healthyServers, rateLimiter); err != nil {
					log.Printf("Failed to sync A records: %v", err)
				}
				txtHost := "health." + e.Hostname
				txtContent := strings.Join(healthyServers, ",")
				if err := cfProvider.SyncTXTRecord(ctx, zoneID, txtHost, txtContent, rateLimiter); err != nil {
					log.Printf("Failed to sync TXT record: %v", err)
				}
			}
		}

		fmt.Println("Waiting 60 seconds before next cycle...")
		time.Sleep(60 * time.Second)
	}
}
