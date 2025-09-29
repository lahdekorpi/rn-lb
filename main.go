package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"gopkg.in/yaml.v3"
)

type CloudflareConfig struct {
	CFAPIToken  string `yaml:"cf_api_token"`
	CFAccountID string `yaml:"cf_account_id"`
	CFZoneID    string `yaml:"cf_zone_id"`
}

type GlobalConfig struct {
	Timeout       int              `yaml:"timeout"`
	Retries       int              `yaml:"retries"`
	RetryWait     int              `yaml:"retry_wait"`
	CheckInterval int              `yaml:"check_interval"`
	Cloudflare    CloudflareConfig `yaml:"cloudflare"`
}

type EntityConfig struct {
	Name       string           `yaml:"name"`
	Hostname   string           `yaml:"hostname"`
	Servers    []string         `yaml:"servers"`
	Timeout    int              `yaml:"timeout,omitempty"`
	Retries    int              `yaml:"retries,omitempty"`
	RetryWait  int              `yaml:"retry_wait,omitempty"`
	Cloudflare CloudflareConfig `yaml:"cloudflare,omitempty"`
}

type Config struct {
	Global   GlobalConfig   `yaml:"global"`
	Entities []EntityConfig `yaml:"entities"`
}

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
	for i := range cfg.Entities {
		e := &cfg.Entities[i]
		if e.Timeout == 0 {
			e.Timeout = cfg.Global.Timeout
		}
		if e.Retries == 0 {
			e.Retries = cfg.Global.Retries
		}
		if e.RetryWait == 0 {
			e.RetryWait = cfg.Global.RetryWait
		}
		// Täydennetään myös Cloudflare-arvot globaalista jos puuttuu
		if e.Cloudflare.CFAPIToken == "" {
			e.Cloudflare.CFAPIToken = cfg.Global.Cloudflare.CFAPIToken
		}
		if e.Cloudflare.CFAccountID == "" {
			e.Cloudflare.CFAccountID = cfg.Global.Cloudflare.CFAccountID
		}
	}
}

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
	log.Printf("  [healthcheck] %s FAIL: %v", url, lastErr)
	return false
}

func processEntity(ctx context.Context, cfAPI *cloudflare.API, e EntityConfig) {
	log.Printf("[entity:%s] checking servers...", e.Name)

	var wg sync.WaitGroup
	results := make(chan string, len(e.Servers))

	for _, server := range e.Servers {
		wg.Add(1)
		go func(srv string) {
			defer wg.Done()
			url := srv
			if !(len(srv) > 7 && (srv[:7] == "http://" || srv[:8] == "https://")) {
				url = "http://" + srv
			}
			if healthCheck(url, e.Timeout, e.Retries, e.RetryWait) {
				results <- srv
			}
		}(server)
	}

	wg.Wait()
	close(results)

	var healthy []string
	for ip := range results {
		healthy = append(healthy, ip)
	}

	if len(healthy) == 0 {
		log.Printf("[entity:%s] no healthy servers, skipping DNS update", e.Name)
		return
	}

	zoneID := e.Cloudflare.CFZoneID
	if zoneID == "" {
		log.Printf("[entity:%s] missing Cloudflare zone ID", e.Name)
		return
	}

	zone := cloudflare.ZoneIdentifier(zoneID)

	// Hae olemassa olevat A-recordit
	records, _, err := cfAPI.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: e.Hostname,
	})
	if err != nil {
		log.Printf("[entity:%s] failed to list DNS records: %v", e.Name, err)
		return
	}

	currentIPs := make(map[string]struct{})
	for _, r := range records {
		currentIPs[r.Content] = struct{}{}
	}

	// Tarkista onko muutos tarpeen
	needUpdate := false
	if len(currentIPs) != len(healthy) {
		needUpdate = true
	} else {
		for _, ip := range healthy {
			if _, ok := currentIPs[ip]; !ok {
				needUpdate = true
				break
			}
		}
	}

	if !needUpdate {
		log.Printf("[entity:%s] DNS already up-to-date", e.Name)
		return
	}

	// Poista vanhat recordit
	for _, r := range records {
		err := cfAPI.DeleteDNSRecord(ctx, zone, r.ID)
		if err != nil {
			log.Printf("[entity:%s] failed to delete record %s: %v", e.Name, r.Content, err)
		}
	}

	// Lisää uudet recordit
	for _, ip := range healthy {
		_, err := cfAPI.CreateDNSRecord(ctx, zone, cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    e.Hostname,
			Content: ip,
			TTL:     60,
		})
		if err != nil {
			log.Printf("[entity:%s] failed to create record for %s: %v", e.Name, ip, err)
		} else {
			log.Printf("[entity:%s] added A-record %s -> %s", e.Name, e.Hostname, ip)
		}
	}
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	cfg.ApplyDefaults()

	ctx := context.Background()

	cfAPI, err := cloudflare.NewWithAPIToken(cfg.Global.Cloudflare.CFAPIToken)
	if err != nil {
		log.Fatalf("failed to init Cloudflare API: %v", err)
	}

	interval := cfg.Global.CheckInterval
	if interval == 0 {
		interval = 5
	}

	for {
		log.Println("=== New health check round ===")

		for _, e := range cfg.Entities {
			processEntity(ctx, cfAPI, e)
		}

		log.Printf("Waiting %d seconds before next round...", interval)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
