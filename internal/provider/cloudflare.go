package provider

import (
	"context"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// CloudflareProvider on provider-toteutus, joka käyttää Cloudflaren APIa.
type CloudflareProvider struct {
	api       *cloudflare.API
	accountID string
	zoneID    string
}

// NewCloudflareProvider luo uuden CloudflareProviderin.
func NewCloudflareProvider(apiToken, accountID, zoneID string) (*CloudflareProvider, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: init failed: %w", err)
	}
	return &CloudflareProvider{
		api:       api,
		accountID: accountID,
		zoneID:    zoneID,
	}, nil
}

// ---- A-record operations ----

// GetRecord hakee ensimmäisen A-tietueen sisällön annetulle hostname:lle.
// Jos tietuetta ei löydy, palautetaan tyhjä string ja error nil (käyttäjän logiikka voi tulkita).
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

// UpdateRecord päivittää olemassa olevan A-tietueen tai luo uuden, jos ei löydy.
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

	// Luo jos ei ole olemassa
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
		return nil
	}

	// Päivitä ensimmäinen löytyvä
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
	return nil
}

// ---- TXT-record operations (leader election) ----

// GetTXT palauttaa TXT-tietueen sisällön (ensimmäinen löytyvä) tai tyhjän stringin, jos ei löydy.
// Ei palauta erroria, paitsi jos API-kutsu epäonnistuu.
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
	// Cloudflare palauttaa Content kentässä usein lainausmerkeissä — caller voi käsitellä tai ne voi jättää.
	return records[0].Content, nil
}

// UpdateTXT luo tai päivittää TXT-tietueen. Käyttötarkoitus: election-lease (esim. "node-id|expiryUnix").
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

	// Jos ei löydy, luodaan
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

	// Päivitetään ensimmäinen löytyvä
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

// compile-time check että CloudflareProvider implementoi Provider-rajapinnan
var _ Provider = (*CloudflareProvider)(nil)
