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

// getConfigValue hakee arvon entityltä ensin, fallback globaalista
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
	log.Printf("Server %s ei vastannut: %v", url, lastErr)
	return false
}

// Päivitä DNS Cloudflaren kautta terveillä IP:illä
func updateDNSRecords(cfAPI *cloudflare.API, zoneID, hostname string, healthyIPs []string) error {
	ctx := context.Background()

	// ResourceContainer zone-ID:llä
	rc := cloudflare.ZoneIdentifier(zoneID)

	// hae nykyiset A-recordit
	recs, _, err := cfAPI.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("DNS recordien haku epäonnistui: %w", err)
	}

	// poista vanhat
	for _, r := range recs {
		err := cfAPI.DeleteDNSRecord(ctx, rc, r.ID)
		if err != nil {
			return fmt.Errorf("recordin poisto epäonnistui (%s): %w", r.Name, err)
		}
	}

	// luo uudet
	for _, ip := range healthyIPs {
		newRec := cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    hostname,
			Content: ip,
			TTL:     60,
		}
		_, err := cfAPI.CreateDNSRecord(ctx, rc, newRec)
		if err != nil {
			return fmt.Errorf("recordin luonti epäonnistui (%s -> %s): %w", hostname, ip, err)
		}
	}

	return nil
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Virhe configin latauksessa: %v", err)
	}
	cfg.ApplyDefaults()

	fmt.Printf("Entities: %d\n", len(cfg.Entities))

	// Cloudflare client globaalilla tokenilla
	globalToken := getConfigValue(cfg, "", "cf_api_token")
	cfAPI, err := cloudflare.NewWithAPIToken(globalToken)
	if err != nil {
		log.Fatal("Cloudflare clientin luonti epäonnistui:", err)
	}

	for {
		fmt.Println("\nUusi kierros:", time.Now().Format("15:04:05"))

		for _, e := range cfg.Entities {
			zoneID := getConfigValue(cfg, e.Name, "cf_zone_id")
			fmt.Printf("\nEntity: %s (zone: %s, hostname: %s)\n", e.Name, zoneID, e.Hostname)

			var healthy []string
			for _, server := range e.Servers {
				url := server
				if !(len(server) > 7 && (server[:7] == "http://" || server[:8] == "https://")) {
					url = "http://" + server
				}
				if healthCheck(url, e.Timeout, e.Retries, e.RetryWait) {
					fmt.Printf("Server %s vastasi OK\n", server)
					healthy = append(healthy, server)
				}
			}

			if len(healthy) > 0 {
				err := updateDNSRecords(cfAPI, zoneID, e.Hostname, healthy)
				if err != nil {
					log.Printf("DNS päivitys epäonnistui entitylle %s: %v", e.Name, err)
				} else {
					fmt.Printf("DNS päivitetty: %s -> %v\n", e.Hostname, healthy)
				}
			} else {
				log.Printf("Ei terveitä servereitä entitylle %s, DNS:ää ei päivitetä", e.Name)
			}
		}

		fmt.Println("Odotetaan 5 sekuntia ennen seuraavaa kierrosta...")
		time.Sleep(5 * time.Second)
	}
}
