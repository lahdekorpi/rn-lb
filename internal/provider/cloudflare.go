// internal/provider/cloudflare.go
package provider

import (
	"fmt"
	"log"

	cf "github.com/cloudflare/cloudflare-go"
)

// CloudflareProvider toteuttaa Provider-rajapinnan
type CloudflareProvider struct {
	client  *cf.API
	account string
	zone    string
}

var _ Provider = (*CloudflareProvider)(nil)

func NewCloudflareProvider(apiToken, accountID, zoneID string) (*CloudflareProvider, error) {
	client, err := cf.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare client: %w", err)
	}
	return &CloudflareProvider{
		client:  client,
		account: accountID,
		zone:    zoneID,
	}, nil
}

func (p *CloudflareProvider) UpdateARecord(host string, ips []string, proxied bool, ttl int) error {
	log.Printf("[Cloudflare] Update A record: host=%s ips=%v proxied=%v ttl=%d",
		host, ips, proxied, ttl)
	return nil
}

func (p *CloudflareProvider) UpdateTXTRecord(host string, records []string, ttl int) error {
	log.Printf("[Cloudflare] Update TXT record: host=%s records=%v ttl=%d",
		host, records, ttl)
	return nil
}
