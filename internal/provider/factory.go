// internal/provider/factory.go
package provider

import (
	"fmt"

	"rn-lb/internal/config"
)

// NewProvider palauttaa Provider-instanssin konfiguraation perusteella.
func NewProvider(cfg config.ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case "cloudflare":
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
