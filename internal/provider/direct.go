package provider

import "log"

// DirectProvider on yksinkertainen stubi, joka ei tee oikeita DNS-kutsuja.
type DirectProvider struct{}

func NewDirectProvider() *DirectProvider {
	return &DirectProvider{}
}

func (p *DirectProvider) UpdateRecord(hostname string, value string, proxied bool, ttl int) error {
	log.Printf("[direct] A UPDATE %s → %s (TTL=%d, proxied=%v)", hostname, value, ttl, proxied)
	return nil
}

func (p *DirectProvider) GetRecord(hostname string) (string, error) {
	log.Printf("[direct] A GET %s", hostname)
	return "", nil
}

// TXT-recordit leader electionia varten
func (p *DirectProvider) GetTXT(hostname string) (string, error) {
	log.Printf("[direct] TXT GET %s", hostname)
	return "", nil
}

func (p *DirectProvider) UpdateTXT(hostname, value string, ttl int) error {
	log.Printf("[direct] TXT UPDATE %s → %s (TTL=%d)", hostname, value, ttl)
	return nil
}
