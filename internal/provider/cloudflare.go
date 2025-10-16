package provider

import (
	"context"
	"fmt"

	cf "github.com/cloudflare/cloudflare-go"
)

// CloudflareProvider hallitsee DNS-tietueita Cloudflare API:n kautta.
type CloudflareProvider struct {
	api      *cf.API
	zoneID   string
	account  string
	apiToken string
}

// NewCloudflareProvider alustaa uuden Cloudflare-providerin.
func NewCloudflareProvider(apiToken, accountID, zoneID string) (*CloudflareProvider, error) {
	api, err := cf.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare client: %w", err)
	}

	return &CloudflareProvider{
		api:      api,
		zoneID:   zoneID,
		account:  accountID,
		apiToken: apiToken,
	}, nil
}

// GetRecord hakee tietyn A- tai CNAME-tietueen.
func (p *CloudflareProvider) GetRecord(hostname string) (string, error) {
	ctx := context.Background()
	rc := cf.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, rc, cf.ListDNSRecordsParams{
		Name: hostname,
		Type: "A",
	})
	if err != nil {
		return "", fmt.Errorf("cloudflare: list records failed: %w", err)
	}
	if len(records) == 0 {
		return "", fmt.Errorf("cloudflare: record not found for %s", hostname)
	}

	return records[0].Content, nil
}

// UpdateRecord päivittää tai luo DNS-tietueen.
func (p *CloudflareProvider) UpdateRecord(hostname string, value string, proxied bool, ttl int) error {
	ctx := context.Background()
	rc := cf.ZoneIdentifier(p.zoneID)

	// Tarkistetaan, onko tietue olemassa
	records, _, err := p.api.ListDNSRecords(ctx, rc, cf.ListDNSRecordsParams{
		Name: hostname,
		Type: "A",
	})
	if err != nil {
		return fmt.Errorf("cloudflare: list records failed: %w", err)
	}

	if len(records) == 0 {
		// Luo uusi tietue
		_, err := p.api.CreateDNSRecord(ctx, rc, cf.CreateDNSRecordParams{
			Type:    "A",
			Name:    hostname,
			Content: value,
			TTL:     ttl,
			Proxied: &proxied,
		})
		if err != nil {
			return fmt.Errorf("cloudflare: create record failed: %w", err)
		}
		return nil
	}

	// Päivitä olemassa oleva tietue
	rec := records[0]
	_, err = p.api.UpdateDNSRecord(ctx, rc, cf.UpdateDNSRecordParams{
		ID:      rec.ID,
		Type:    "A",
		Name:    hostname,
		Content: value,
		TTL:     ttl,
		Proxied: &proxied,
	})
	if err != nil {
		return fmt.Errorf("cloudflare: update record failed: %w", err)
	}
	return nil
}

// Compile-time check: varmistaa että tämä struct implementoi Provider-rajapinnan.
var _ Provider = (*CloudflareProvider)(nil)
