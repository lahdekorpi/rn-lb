package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	Timeout   int `yaml:"timeout"`
	Retries   int `yaml:"retries"`
	RetryWait int `yaml:"retry_wait"`
}

type EntityConfig struct {
	Name      string   `yaml:"name"`
	Servers   []string `yaml:"servers"`
	Timeout   int      `yaml:"timeout,omitempty"`
	Retries   int      `yaml:"retries,omitempty"`
	RetryWait int      `yaml:"retry_wait,omitempty"`
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

// HTTP health check
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

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Virhe configin latauksessa: %v", err)
	}
	cfg.ApplyDefaults()

	fmt.Println("Global config:")
	fmt.Printf("Timeout=%d ms, Retries=%d, RetryWait=%d ms\n",
		cfg.Global.Timeout, cfg.Global.Retries, cfg.Global.RetryWait)

	// Loopataan jatkuvasti
	for {
		fmt.Println("\nUusi kierros:", time.Now().Format("15:04:05"))

		for _, e := range cfg.Entities {
			fmt.Printf("\nEntity: %s\n", e.Name)
			for _, server := range e.Servers {
				ok := healthCheck(server, e.Timeout, e.Retries, e.RetryWait)
				if ok {
					fmt.Printf("Server %s vastasi OK\n", server)
				}
			}
		}

		fmt.Println("Odotetaan 5 sekuntia ennen seuraavaa kierrosta...")
		time.Sleep(5 * time.Second) // <- vaihdettu 5 sekuntiin
	}

}
