package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ---------- Root Config ----------

type Config struct {
	Daemon   DaemonConfig   `yaml:"daemon"`
	Global   GlobalConfig   `yaml:"global"`
	Entities []EntityConfig `yaml:"entities"`
}

// ---------- Daemon ----------

type DaemonConfig struct {
	ID        string `yaml:"id"`
	IDFile    string `yaml:"id_file"`
	Priority  int    `yaml:"priority"`
	DNSServer string `yaml:"dns_server"`

	Isolation IsolationConfig `yaml:"isolation"`
}

type IsolationConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Proxies       []ProxyConfig `yaml:"proxies"`
	JWTSecretFile string        `yaml:"jwt_secret_file"`
	JWTTTL        time.Duration `yaml:"jwt_ttl"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryCount    int           `yaml:"retry_count"`
}

type ProxyConfig struct {
	URL      string `yaml:"url"`
	Priority int    `yaml:"priority"`
}

// ---------- Global ----------

type GlobalConfig struct {
	Health    HealthGlobalConfig    `yaml:"health"`
	Consensus ConsensusGlobalConfig `yaml:"consensus"`
	Election  ElectionGlobalConfig  `yaml:"election"`
	Provider  ProviderConfig        `yaml:"provider"`
}

type HealthGlobalConfig struct {
	CheckInterval time.Duration `yaml:"check_interval"`
	Timeout       time.Duration `yaml:"timeout"`
	Retries       int           `yaml:"retries"`
	RetryWait     time.Duration `yaml:"retry_wait"`
}

type ConsensusGlobalConfig struct {
	CoordinationInterval time.Duration `yaml:"coordination_interval"`
	MinReporters         int           `yaml:"min_reporters"`
	Threshold            float64       `yaml:"threshold"`
	IsolationDetection   bool          `yaml:"isolation_detection"`
	ReportTTL            time.Duration `yaml:"report_ttl"`
	AllDownProtection    bool          `yaml:"all_down_protection"`
}

type ElectionGlobalConfig struct {
	LeaseDuration     time.Duration `yaml:"lease_duration"`
	GracePeriod       time.Duration `yaml:"grace_period"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

// ---------- Provider ----------

type ProviderConfig struct {
	Type       string           `yaml:"type"`
	Cloudflare CloudflareConfig `yaml:"cloudflare"`
}

type CloudflareConfig struct {
	APIToken  string `yaml:"api_token"`
	AccountID string `yaml:"account_id"`
	ZoneID    string `yaml:"zone_id"`
}

// ---------- Entities ----------

type EntityConfig struct {
	Name      string          `yaml:"name"`
	Hostname  string          `yaml:"hostname"`
	Servers   []ServerEntry   `yaml:"servers"`
	DNS       DNSConfig       `yaml:"dns"`
	Health    HealthConfig    `yaml:"health"`
	Consensus ConsensusConfig `yaml:"consensus"`
	Provider  ProviderConfig  `yaml:"provider"`
}

type ServerEntry struct {
	ID      string     `yaml:"id"`
	Address string     `yaml:"address"`
	Check   CheckEntry `yaml:"check"`
}

type CheckEntry struct {
	Type        string        `yaml:"type"`               // "http" or "tcp"
	Protocol    string        `yaml:"protocol,omitempty"` // for http: "http"/"https"
	Port        int           `yaml:"port"`
	Path        string        `yaml:"path,omitempty"`
	Method      string        `yaml:"method,omitempty"`
	ValidStatus []int         `yaml:"valid_status,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty"`
}

type DNSConfig struct {
	TTL     int  `yaml:"ttl"`
	Proxied bool `yaml:"proxied"`
}

type HealthConfig struct {
	Timeout time.Duration `yaml:"timeout"`
}

type ConsensusConfig struct {
	Threshold float64 `yaml:"threshold"`
}

// ---------- Loader ----------

func LoadConfig(path string) (*Config, error) {
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
