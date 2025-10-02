package provider

import (
	cloudflare "github.com/cloudflare/cloudflare-go"
)

func NewClient(token string) (*cloudflare.API, error) {
	return cloudflare.NewWithAPIToken(token)
}
