package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	Timeout   int `yaml:"timeout"`
	Retries   int `yaml:"retries"`
	RetryWait int `yaml:"retry_wait"`
}

type EntityConfig struct {
	Name      string   `yaml:"name"`
	Timeout   int      `yaml:"timeout,omitempty"`
	Retries   int      `yaml:"retries,omitempty"`
	RetryWait int      `yaml:"retry_wait,omitempty"`
	Servers   []string `yaml:"servers"`
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

// Täydennä globaalit arvot entiteeteille jos niiltä puuttuu
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

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Virhe configin latauksessa: %v", err)
	}

	cfg.ApplyDefaults()

	fmt.Println("🌍 Global config:")
	fmt.Printf("Timeout: %d ms, Retries: %d, RetryWait: %d ms\n",
		cfg.Global.Timeout, cfg.Global.Retries, cfg.Global.RetryWait)

	fmt.Println("\n📦 Entities:")
	for _, e := range cfg.Entities {
		fmt.Printf("Entity: %s, Timeout: %d ms, Retries: %d, RetryWait: %d ms, Servers: %v\n",
			e.Name, e.Timeout, e.Retries, e.RetryWait, e.Servers)
	}
}
