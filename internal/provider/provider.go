package provider

// Provider määrittelee yleisen DNS-provider-rajapinnan
type Provider interface {
	// A-recordit
	UpdateRecord(hostname string, value string, proxied bool, ttl int) error
	GetRecord(hostname string) (string, error)

	// TXT-recordit leader electionia varten
	GetTXT(hostname string) (string, error)
	UpdateTXT(hostname string, value string, ttl int) error
}
