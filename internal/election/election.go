package election

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"rn-lb/internal/config"
	"rn-lb/internal/provider"
)

// Simple TXT format: "<node-id>|<expiryUnix>"
// e.g. "node-1|1700000000"

// Elector handles per-entity election loop.
type Elector struct {
	Entity        config.EntityConfig
	Provider      provider.Provider
	MyID          string
	LeaseDuration time.Duration
	Heartbeat     time.Duration
	TXTName       string // e.g. "leader.<hostname>" or configurable
	TTL           int
	cancelCtx     context.Context
	cancelFunc    context.CancelFunc
}

// NewElector creates elector for an entity. TXT name defaults to "leader.<hostname>".
func NewElector(ent config.EntityConfig, p provider.Provider, myID string, lease, heartbeat time.Duration) *Elector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Elector{
		Entity:        ent,
		Provider:      p,
		MyID:          myID,
		LeaseDuration: lease,
		Heartbeat:     heartbeat,
		TXTName:       "leader." + ent.Hostname,
		TTL:           int(lease.Seconds()), // use lease seconds as TTL
		cancelCtx:     ctx,
		cancelFunc:    cancel,
	}
}

// Run runs the election loop and sends leader state on the channel (true if leader).
func (e *Elector) Run(leaderCh chan<- bool) {
	ticker := time.NewTicker(e.Heartbeat)
	defer ticker.Stop()
	// initial check immediately
	e.tryAcquireOnce(leaderCh)
	for {
		select {
		case <-e.cancelCtx.Done():
			return
		case <-ticker.C:
			e.tryAcquireOnce(leaderCh)
		}
	}
}

// Stop stops the elector loop.
func (e *Elector) Stop() {
	e.cancelFunc()
}

// tryAcquireOnce reads TXT and tries to acquire/refresh lease.
func (e *Elector) tryAcquireOnce(leaderCh chan<- bool) {
	now := time.Now().Unix()
	txt, err := e.Provider.GetTXT(e.TXTName)
	if err != nil {
		// on error, assume not leader
		leaderCh <- false
		return
	}

	ownerID, expiry := parseTXT(txt)
	if ownerID == "" || expiry <= now {
		// try to write own lease
		newExpiry := time.Now().Add(e.LeaseDuration).Unix()
		content := fmt.Sprintf("%s|%d", e.MyID, newExpiry)
		// attempt update
		if err := e.Provider.UpdateTXT(e.TXTName, content, e.TTL); err == nil {
			leaderCh <- true
			return
		}
		leaderCh <- false
		return
	}

	// currently owned by someone else and still valid
	if ownerID == e.MyID {
		// refresh lease by updating expiry
		newExpiry := time.Now().Add(e.LeaseDuration).Unix()
		content := fmt.Sprintf("%s|%d", e.MyID, newExpiry)
		_ = e.Provider.UpdateTXT(e.TXTName, content, e.TTL) // ignore error; still leader if ownerID==MyID
		leaderCh <- true
		return
	}

	leaderCh <- false
}

// parseTXT returns ownerID and expiryUnix; empty txt -> ("",0)
func parseTXT(txt string) (string, int64) {
	if txt == "" {
		return "", 0
	}
	parts := strings.Split(txt, "|")
	if len(parts) < 2 {
		return "", 0
	}
	expiry, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return "", 0
	}
	return strings.TrimSpace(parts[0]), expiry
}
