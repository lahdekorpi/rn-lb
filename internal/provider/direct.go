package provider

import "log"

// DirectProvider on yksinkertainen stubi, joka ei tee oikeita DNS-kutsuja.
// Tätä käytetään testauksessa ja kehityksessä ilman ulkoista API:ta.
type DirectProvider struct{}

func NewDirectProvider() *DirectProvider {
	return &DirectProvider{}
}

func (p *DirectProvider) UpdateRecord(hostname string, value string, proxied bool, ttl int) error {
	log.Printf("[direct] would update record %s → %s (ttl=%d, proxied=%v)", hostname, value, ttl, proxied)
	return nil
}

func (p *DirectProvider) GetRecord(hostname string) (string, error) {
	log.Printf("[direct] would get record for %s", hostname)
	return "127.0.0.1", nil
}
