package provider

import (
	"context"
	"fmt"
	"sync"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

type CloudflareProvider struct {
	api       *cloudflare.API
	accountID string
	zoneID    string

	mu    sync.Mutex
	cache map[string]string
}

// ---------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------

func NewCloudflareProvider(apiToken, accountID, zoneID string) (*CloudflareProvider, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: init failed: %w", err)
	}

	return &CloudflareProvider{
		api:       api,
		accountID: accountID,
		zoneID:    zoneID,
		cache:     make(map[string]string),
	}, nil
}

// ---------------------------------------------------------------------
// A record operations
// ---------------------------------------------------------------------

func (p *CloudflareProvider) GetRecord(hostname string) (string, error) {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return "", fmt.Errorf("cloudflare: list A records failed: %w", err)
	}
	if len(records) == 0 {
		return "", nil
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
		return fmt.Errorf("cloudflare: list A records failed: %w", err)
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
			return fmt.Errorf("cloudflare: create A record failed: %w", err)
		}
		p.SetLastIP(hostname, value)
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
		return fmt.Errorf("cloudflare: update A record failed: %w", err)
	}

	p.SetLastIP(hostname, value)
	return nil
}

// ---------------------------------------------------------------------
// TXT record operations
// ---------------------------------------------------------------------

func (p *CloudflareProvider) GetTXT(hostname string) (string, error) {
	ctx := context.Background()
	zone := cloudflare.ZoneIdentifier(p.zoneID)

	records, _, err := p.api.ListDNSRecords(ctx, zone, cloudflare.ListDNSRecordsParams{
		Type: "TXT",
		Name: hostname,
	})
	if err != nil {
		return "", fmt.Errorf("cloudflare: list TXT records failed: %w", err)
	}
	if len(records) == 0 {
		return "", nil
	}
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
		return fmt.Errorf("cloudflare: list TXT records failed: %w", err)
	}

	if len(records) == 0 {
		_, err := p.api.CreateDNSRecord(ctx, zone, cloudflare.CreateDNSRecordParams{
			Type:    "TXT",
			Name:    hostname,
			Content: value,
			TTL:     ttl,
		})
		if err != nil {
			return fmt.Errorf("cloudflare: create TXT record failed: %w", err)
		}
		return nil
	}

	rec := records[0]
	_, err = p.api.UpdateDNSRecord(ctx, zone, cloudflare.UpdateDNSRecordParams{
		ID:      rec.ID,
		Type:    "TXT",
		Name:    hostname,
		Content: value,
		TTL:     ttl,
	})
	if err != nil {
		return fmt.Errorf("cloudflare: update TXT record failed: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------
// Simple in-process IP cache
// ---------------------------------------------------------------------

func (p *CloudflareProvider) GetLastIP(host string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cache[host]
}

func (p *CloudflareProvider) SetLastIP(host, ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[host] = ip
}

// ensure Provider interface implemented
var _ Provider = (*CloudflareProvider)(nil)
