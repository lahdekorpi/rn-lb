package provider

// Provider määrittelee yhteisen rajapinnan eri DNS-providereille.
// Esim. Cloudflare, Route53, DigitalOcean, jne.
type Provider interface {
	// UpdateARecord päivittää hostille uudet A-recordit (IP-osoitteet).
	UpdateARecord(host string, ips []string, proxied bool, ttl int) error

	// UpdateTXTRecord päivittää hostille uudet TXT-recordit.
	UpdateTXTRecord(host string, records []string, ttl int) error
}
