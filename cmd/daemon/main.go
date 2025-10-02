package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"gopkg.in/yaml.v3"
)

// ---------- CONFIG STRUCTS ----------

type CloudflareConfig struct {
	CFAPIToken  string `yaml:"cf_api_token"`
	CFAccountID string `yaml:"cf_account_id"`
	CFzoneID    string `yaml:"cf_zone_id"`
}

type GlobalConfig struct {
	Timeout    int              `yaml:"timeout"`
	Retries    int              `yaml:"retries"`
	RetryWait  int              `yaml:"retry_wait"`
	Cloudflare CloudflareConfig `yaml:"cloudflare"`
}

type EntityConfig struct {
	Name             string           `yaml:"name"`
	Hostname         string           `yaml:"hostname"`
	Servers          []string         `yaml:"servers"`
	Timeout          int              `yaml:"timeout,omitempty"`
	Retries          int              `yaml:"retries,omitempty"`
	RetryWait        int              `yaml:"retry_wait,omitempty"`
	CloudflareConfig CloudflareConfig `yaml:"cloudflare,omitempty"`
}

type Config struct {
	Global   GlobalConfig   `yaml:"global"`
	Entities []EntityConfig `yaml:"entities"`
}

// ---------- CONFIG LOADING ----------

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (cfg *Config) ApplyDefaults() {
	for i, e := range cfg.Entities {
		if e.Timeout == 0 {
			cfg.Entities[i].Timeout = cfg.Global.Timeout
		}
		if e.Retries == 0 {
			cfg.Entities[i].Retries = cfg.Global.Retries
		}
		if e.RetryWait == 0 {
			cfg.Entities[i].RetryWait = cfg.Global.RetryWait
		}
	}
}

// ---------- CONFIG HELPER ----------

func getConfigValue(cfg *Config, entityName string, key string) string {
	var entity *EntityConfig
	for _, e := range cfg.Entities {
		if e.Name == entityName {
			entity = &e
			break
		}
	}

	if entity != nil {
		switch key {
		case "timeout":
			if entity.Timeout != 0 {
				return fmt.Sprintf("%d", entity.Timeout)
			}
		case "retries":
			if entity.Retries != 0 {
				return fmt.Sprintf("%d", entity.Retries)
			}
		case "retry_wait":
			if entity.RetryWait != 0 {
				return fmt.Sprintf("%d", entity.RetryWait)
			}
		case "cf_zone_id":
			if entity.CloudflareConfig.CFzoneID != "" {
				return entity.CloudflareConfig.CFzoneID
			}
		case "cf_api_token":
			if entity.CloudflareConfig.CFAPIToken != "" {
				return entity.CloudflareConfig.CFAPIToken
			}
		case "cf_account_id":
			if entity.CloudflareConfig.CFAccountID != "" {
				return entity.CloudflareConfig.CFAccountID
			}
		}
	}

	switch key {
	case "timeout":
		return fmt.Sprintf("%d", cfg.Global.Timeout)
	case "retries":
		return fmt.Sprintf("%d", cfg.Global.Retries)
	case "retry_wait":
		return fmt.Sprintf("%d", cfg.Global.RetryWait)
	case "cf_api_token":
		return cfg.Global.Cloudflare.CFAPIToken
	case "cf_account_id":
		return cfg.Global.Cloudflare.CFAccountID
	case "cf_zone_id":
		return cfg.Global.Cloudflare.CFzoneID
	}

	return ""
}

// ---------- HEALTH CHECK ----------

func healthCheck(url string, timeout, retries, retryWait int) bool {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		resp, err := client.Get(url)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return true
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(time.Duration(retryWait) * time.Millisecond)
	}
	log.Printf("Server %s did not respond: %v", url, lastErr)
	return false
}

// ---------- DNS HELPERS ----------

func getTXTRecord(hostname string) ([]string, error) {
	records, err := net.LookupTXT(hostname)
	if err != nil {
		return nil, err
	}
	if len(records) > 0 {
		return strings.Split(records[0], ","), nil
	}
	return []string{}, nil
}

func syncDNSRecords(ctx context.Context, api *cloudflare.API, zoneID, hostname string, healthyServers []string, rateLimiter <-chan time.Time) error {
	rc := cloudflare.ZoneIdentifier(zoneID)

	existing, _, err := api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("failed to list DNS records: %w", err)
	}

	desired := make(map[string]bool)
	for _, ip := range healthyServers {
		desired[ip] = true
	}

	for _, rec := range existing {
		if !desired[rec.Content] {
			<-rateLimiter
			log.Printf("Deleting old A record %s -> %s", rec.Name, rec.Content)
			if err := api.DeleteDNSRecord(ctx, rc, rec.ID); err != nil {
				return fmt.Errorf("failed to delete DNS record: %w", err)
			}
		} else {
			delete(desired, rec.Content)
		}
	}

	for ip := range desired {
		<-rateLimiter
		params := cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    hostname,
			Content: ip,
			TTL:     60,
		}
		log.Printf("Creating A record %s -> %s", hostname, ip)
		if _, err := api.CreateDNSRecord(ctx, rc, params); err != nil {
			return fmt.Errorf("failed to create DNS record: %w", err)
		}
	}

	return nil
}

func syncTXTRecord(ctx context.Context, api *cloudflare.API, zoneID, hostname, content string, rateLimiter <-chan time.Time) error {
	rc := cloudflare.ZoneIdentifier(zoneID)

	existing, _, err := api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Type: "TXT",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("failed to list TXT records: %w", err)
	}

	for _, rec := range existing {
		if rec.Content != content {
			<-rateLimiter
			log.Printf("Deleting old TXT record %s -> %s", rec.Name, rec.Content)
			if err := api.DeleteDNSRecord(ctx, rc, rec.ID); err != nil {
				return fmt.Errorf("failed to delete TXT record: %w", err)
			}
		} else {
			return nil
		}
	}

	<-rateLimiter
	params := cloudflare.CreateDNSRecordParams{
		Type:    "TXT",
		Name:    hostname,
		Content: content,
		TTL:     60,
	}
	log.Printf("Creating TXT record %s -> %s", hostname, content)
	if _, err := api.CreateDNSRecord(ctx, rc, params); err != nil {
		return fmt.Errorf("failed to create TXT record: %w", err)
	}

	return nil
}

// ---------- HELPER ----------

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------- MAIN ----------

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	cfg.ApplyDefaults()

	fmt.Printf("Entities: %d\n", len(cfg.Entities))

	globalToken := getConfigValue(cfg, "", "cf_api_token")
	cfAPI, err := cloudflare.NewWithAPIToken(globalToken)
	if err != nil {
		log.Fatal("Failed to create Cloudflare client:", err)
	}

	rateLimiter := time.Tick(500 * time.Millisecond)

	for {
		ctx := context.Background()
		fmt.Println("\nNew cycle:", time.Now().Format("15:04:05"))

		for _, e := range cfg.Entities {
			zoneID := getConfigValue(cfg, e.Name, "cf_zone_id")
			fmt.Printf("\nEntity: %s (zone: %s)\n", e.Name, zoneID)

			healthyServers := []string{}
			for _, server := range e.Servers {
				url := server
				if !(strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://")) {
					url = "http://" + server
				}
				if healthCheck(url, e.Timeout, e.Retries, e.RetryWait) {
					fmt.Printf("Server %s responded OK\n", server)
					healthyServers = append(healthyServers, server)
				}
			}

			if len(healthyServers) == 0 {
				fmt.Println("No healthy servers -> clearing A records and marking TXT as DOWN")
				if err := syncDNSRecords(ctx, cfAPI, zoneID, e.Hostname, []string{}, rateLimiter); err != nil {
					log.Printf("Failed to clear A records: %v", err)
				}
				if err := syncTXTRecord(ctx, cfAPI, zoneID, "health."+e.Hostname, "DOWN", rateLimiter); err != nil {
					log.Printf("Failed to update TXT record: %v", err)
				}
			} else {
				if err := syncDNSRecords(ctx, cfAPI, zoneID, e.Hostname, healthyServers, rateLimiter); err != nil {
					log.Printf("Failed to sync A records: %v", err)
				}
				txtHost := "health." + e.Hostname
				txtContent := strings.Join(healthyServers, ",")
				if err := syncTXTRecord(ctx, cfAPI, zoneID, txtHost, txtContent, rateLimiter); err != nil {
					log.Printf("Failed to sync TXT record: %v", err)
				}
			}
		}

		fmt.Println("Waiting 60 seconds before next cycle...")
		time.Sleep(60 * time.Second)
	}
}
