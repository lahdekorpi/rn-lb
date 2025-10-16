package provider

import (
	"fmt"

	"rn-lb/internal/config"
)

// Provider määrittelee yleisen DNS-provider-rajapinnan

// NewProvider palauttaa oikean providerin konfiguraation perusteella.
func NewProvider(cfg config.ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case "cloudflare":
		p, err := NewCloudflareProvider(
			cfg.Cloudflare.APIToken,
			cfg.Cloudflare.AccountID,
			cfg.Cloudflare.ZoneID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Cloudflare provider: %w", err)
		}
		return p, nil

	case "direct":
		return NewDirectProvider(), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
}
