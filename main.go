package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"gopkg.in/yaml.v3"
)

type CloudflareConfig struct {
	CFAPIToken  string `yaml:"cf_api_token"`
	CFAccountID string `yaml:"cf_account_id"`
	CFzoneID    string `yaml:"cf_zone_id"`
}

type GlobalConfig struct {
	Timeout       int              `yaml:"timeout"`
	Retries       int              `yaml:"retries"`
	RetryWait     int              `yaml:"retry_wait"`
	CheckInterval int              `yaml:"check_interval"`
	Cloudflare    CloudflareConfig `yaml:"cloudflare"`
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

// Health check HTTP GET
func healthCheck(url string, timeout, retries, retryWait int) bool {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		resp, err := client.Get("http://" + url) // oletetaan että server on IP/hostname ilman schemeä
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
	log.Printf("Server %s ei vastannut: %v", url, lastErr)
	return false
}

// Synkkaa terveet IP:t Cloudflareen
func syncDNSRecords(cfAPI *cloudflare.API, zoneID, hostname string, healthyIPs []string) error {
	ctx := context.Background()

	// Hae nykyiset DNS-recordit
	records, _, err := cfAPI.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Name: hostname,
		Type: "A",
	})
	if err != nil {
		return fmt.Errorf("DNS-recordien haku epäonnistui: %w", err)
	}

	existingIPs := map[string]string{} // ip -> recordID
	for _, r := range records {
		existingIPs[r.Content] = r.ID
	}

	// Poista epäkelvot recordit
	for ip, recID := range existingIPs {
		if !contains(healthyIPs, ip) {
			log.Printf("Poistetaan epäkelpo DNS A-record: %s -> %s", hostname, ip)
			err := cfAPI.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), recID)
			if err != nil {
				log.Printf("Virhe poistaessa %s: %v", ip, err)
			}
		}
	}

	// Lisää uudet puuttuvat recordit
	for _, ip := range healthyIPs {
		if _, exists := existingIPs[ip]; !exists {
			log.Printf("Lisätään uusi DNS A-record: %s -> %s", hostname, ip)
			_, err := cfAPI.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
				Type:    "A",
				Name:    hostname,
				Content: ip,
				TTL:     60, // lyhyt TTL
			})
			if err != nil {
				log.Printf("Virhe lisättäessä %s: %v", ip, err)
			}
		}
	}

	return nil
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Virhe configin latauksessa: %v", err)
	}
	cfg.ApplyDefaults()

	// Luo Cloudflare client
	cfAPI, err := cloudflare.NewWithAPIToken(cfg.Global.Cloudflare.CFAPIToken)
	if err != nil {
		log.Fatal("Cloudflare clientin luonti epäonnistui:", err)
	}

	for {
		fmt.Println("\nUusi kierros:", time.Now().Format("15:04:05"))

		for _, e := range cfg.Entities {
			var healthy []string
			fmt.Printf("\nEntity: %s (hostname: %s)\n", e.Name, e.Hostname)

			// Health check kaikille servereille
			for _, server := range e.Servers {
				if healthCheck(server, e.Timeout, e.Retries, e.RetryWait) {
					fmt.Printf("Server %s vastasi OK\n", server)
					healthy = append(healthy, server)
				}
			}

			// Synkataan Cloudflareen
			if len(healthy) > 0 {
				err := syncDNSRecords(cfAPI, e.CloudflareConfig.CFzoneID, e.Hostname, healthy)
				if err != nil {
					log.Printf("Virhe synkatessa DNS-recordeja: %v", err)
				}
			} else {
				log.Printf("Ei yhtään tervettä serveriä entitylle %s", e.Name)
			}
		}

		fmt.Printf("Odotetaan %d sekuntia ennen seuraavaa kierrosta...\n", cfg.Global.CheckInterval)
		time.Sleep(time.Duration(cfg.Global.CheckInterval) * time.Second)
	}
}
