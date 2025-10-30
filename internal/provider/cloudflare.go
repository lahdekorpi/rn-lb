package provider

import (
	"context"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

type CloudflareProvider struct {
	api       *cloudflare.API
	accountID string
	zoneID    string
}

func NewCloudflareProvider(apiToken, accountID, zoneID string) (*CloudflareProvider, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("cloudflare init failed: %w", err)
	}

	return &CloudflareProvider{
		api:       api,
		accountID: accountID,
		zoneID:    zoneID,
	}, nil
}

// GetRecord fetches current A record value
func (p *CloudflareProvider) GetRecord(hostname string) (string, error) {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return "", fmt.Errorf("list dns failed: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("A record not found for %s", hostname)
	}

	return records[0].Content, nil
}

// UpdateRecord updates or creates DNS A record
func (p *CloudflareProvider) UpdateRecord(hostname string, value string, proxied bool, ttl int) error {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	// find existing record
	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("list dns failed: %w", err)
	}

	if len(records) == 0 {
		_, err := p.api.CreateDNSRecord(ctx, zone, cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    hostname,
			Content: value,
			TTL:     ttl,
			Proxied: &proxied,
		})
		if err != nil {
			return fmt.Errorf("create record failed: %w", err)
		}
		return nil
	}

	rec := records[0]

	_, err = p.api.UpdateDNSRecord(ctx, zone, cloudflare.UpdateDNSRecordParams{
		ID:      rec.ID,
		Type:    "A",
		Name:    hostname,
		Content: value,
		TTL:     ttl,
		Proxied: &proxied,
	})
	if err != nil {
		return fmt.Errorf("update record failed: %w", err)
	}
	return nil
}
