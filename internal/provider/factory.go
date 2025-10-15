package provider

import (
	"fmt"
	"rn-lb/internal/config"
)

// NewProvider returns a Provider implementation based on configuration.
func NewProvider(cfg config.ProviderConfig) (interface{}, error) {
	switch cfg.Type {
	case "cloudflare":
		if cfg.Cloudflare.APIToken == "" || cfg.Cloudflare.ZoneID == "" {
			return nil, fmt.Errorf("invalid cloudflare provider config")
		}
		return NewCloudflareProvider(
			cfg.Cloudflare.APIToken,
			cfg.Cloudflare.AccountID,
			cfg.Cloudflare.ZoneID,
		)
	case "direct":
		return NewDirectProvider(), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}
