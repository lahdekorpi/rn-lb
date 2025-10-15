package provider

import (
	"context"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// CloudflareProvider wraps Cloudflare API client.
type CloudflareProvider struct {
	api       *cloudflare.API
	accountID string
	zoneID    string
}

// NewCloudflareProvider creates and validates a new Cloudflare provider.
func NewCloudflareProvider(apiToken, accountID, zoneID string) (*CloudflareProvider, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to init Cloudflare API: %w", err)
	}
	return &CloudflareProvider{
		api:       api,
		accountID: accountID,
		zoneID:    zoneID,
	}, nil
}

// UpdateARecord updates or creates an A record for hostname → ip.
func (p *CloudflareProvider) UpdateARecord(ctx context.Context, hostname, ip string, proxied bool, ttl int) error {
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	// List existing A records for hostname
	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("list DNS records: %w", err)
	}

	// Define base record data
	createParams := cloudflare.CreateDNSRecordParams{
		Type:    "A",
		Name:    hostname,
		Content: ip,
		TTL:     ttl,
		Proxied: &proxied,
	}

	// Create or update
	if len(records) == 0 {
		_, err = p.api.CreateDNSRecord(ctx, zone, createParams)
		if err != nil {
			return fmt.Errorf("create DNS record: %w", err)
		}
	} else {
		updateParams := cloudflare.UpdateDNSRecordParams{
			ID:      records[0].ID,
			Type:    "A",
			Name:    hostname,
			Content: ip,
			TTL:     ttl,
			Proxied: &proxied,
		}
		_, err = p.api.UpdateDNSRecord(ctx, zone, updateParams)
		if err != nil {
			return fmt.Errorf("update DNS record: %w", err)
		}
	}

	return nil
}
