package dns

import (
	"context"
	"fmt"
	"log"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func SyncARecords(ctx context.Context, api *cloudflare.API, zoneID, hostname string, healthyServers []string, rateLimiter <-chan time.Time) error {
	rc := cloudflare.ZoneIdentifier(zoneID)

	existing, _, err := api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Type: "A",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("failed to list DNS records: %w", err)
	}

	desired := make(map[string]bool)
	for _, ip := range healthyServers {
		desired[ip] = true
	}

	for _, rec := range existing {
		if !desired[rec.Content] {
			<-rateLimiter
			log.Printf("Deleting old A record %s -> %s", rec.Name, rec.Content)
			if err := api.DeleteDNSRecord(ctx, rc, rec.ID); err != nil {
				return fmt.Errorf("failed to delete DNS record: %w", err)
			}
		} else {
			delete(desired, rec.Content)
		}
	}

	for ip := range desired {
		<-rateLimiter
		params := cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    hostname,
			Content: ip,
			TTL:     60,
		}
		log.Printf("Creating A record %s -> %s", hostname, ip)
		if _, err := api.CreateDNSRecord(ctx, rc, params); err != nil {
			return fmt.Errorf("failed to create DNS record: %w", err)
		}
	}
	return nil
}

func SyncTXTRecord(ctx context.Context, api *cloudflare.API, zoneID, hostname, content string, rateLimiter <-chan time.Time) error {
	rc := cloudflare.ZoneIdentifier(zoneID)

	existing, _, err := api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Type: "TXT",
		Name: hostname,
	})
	if err != nil {
		return fmt.Errorf("failed to list TXT records: %w", err)
	}

	for _, rec := range existing {
		if rec.Content != content {
			<-rateLimiter
			log.Printf("Deleting old TXT record %s -> %s", rec.Name, rec.Content)
			if err := api.DeleteDNSRecord(ctx, rc, rec.ID); err != nil {
				return fmt.Errorf("failed to delete TXT record: %w", err)
			}
		} else {
			return nil
		}
	}

	<-rateLimiter
	params := cloudflare.CreateDNSRecordParams{
		Type:    "TXT",
		Name:    hostname,
		Content: content,
		TTL:     60,
	}
	log.Printf("Creating TXT record %s -> %s", hostname, content)
	if _, err := api.CreateDNSRecord(ctx, rc, params); err != nil {
		return fmt.Errorf("failed to create TXT record: %w", err)
	}
	return nil
}
