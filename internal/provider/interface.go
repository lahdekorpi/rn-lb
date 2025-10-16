package provider

// Provider määrittelee yleisen DNS-provider-rajapinnan
type Provider interface {
	UpdateRecord(hostname string, value string, proxied bool, ttl int) error
	GetRecord(hostname string) (string, error)
}
