package provider

import "log"

type DirectProvider struct{}

func NewDirectProvider() *DirectProvider {
	return &DirectProvider{}
}

func (p *DirectProvider) UpdateARecord(host string, ips []string, proxied bool, ttl int) error {
	log.Printf("[Direct] Pretend update A record for %s: %v", host, ips)
	return nil
}

func (p *DirectProvider) UpdateTXTRecord(host string, records []string, ttl int) error {
	log.Printf("[Direct] Pretend update TXT record for %s: %v", host, records)
	return nil
}
