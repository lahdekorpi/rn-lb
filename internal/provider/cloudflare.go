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

// ---- A RECORD ----

func (p *CloudflareProvider) GetRecord(hostname string) (string, error) {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return "", fmt.Errorf("A lookup failed: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("A record not found: %s", hostname)
	}

	return records[0].Content, nil
}

func (p *CloudflareProvider) UpdateRecord(hostname string, value string, proxied bool, ttl int) error {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("A list failed: %w", err)
	}

	if len(records) == 0 {
		_, err := p.api.CreateDNSRecord(ctx, zone, cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    hostname,
			Content: value,
			TTL:     ttl,
			Proxied: &proxied,
		})
		return err
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
	return err
}

// ---- TXT RECORD ----

func (p *CloudflareProvider) GetTXT(hostname string) (string, error) {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "TXT",
		Name: hostname,
	})
	if err != nil {
		return "", fmt.Errorf("TXT lookup failed: %w", err)
	}
	if len(records) == 0 {
		return "", nil // return empty string when missing (easier for election logic)
	}

	// records[0].Content typically includes the TXT text
	return records[0].Content, nil
}

func (p *CloudflareProvider) UpdateTXT(hostname, value string, ttl int) error {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "TXT",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("TXT list failed: %w", err)
	}

	// create if missing
	if len(records) == 0 {
		_, err := p.api.CreateDNSRecord(ctx, zone, cloudflare.CreateDNSRecordParams{
			Type:    "TXT",
			Name:    hostname,
			Content: value,
			TTL:     ttl,
		})
		return err
	}

	rec := records[0]
	_, err = p.api.UpdateDNSRecord(ctx, zone, cloudflare.UpdateDNSRecordParams{
		ID:      rec.ID,
		Type:    "TXT",
		Name:    hostname,
		Content: value,
		TTL:     ttl,
	})
	return err
}

// compile-time check
var _ Provider = (*CloudflareProvider)(nil)
