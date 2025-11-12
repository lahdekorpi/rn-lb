package coordinator

import (
	"log"
	"sync"
	"time"

	"rn-lb/internal/config"
	"rn-lb/internal/election"
	"rn-lb/internal/health"
	"rn-lb/internal/provider"
)

// Run starts coordinator loop. It spawns per-entity elector and runs health checks.
// stop channel stops coordinator and all electors.
func Run(cfg *config.Config, p provider.Provider, myID string, stop <-chan struct{}) {
	log.Println("Coordinator starting...")

	var wg sync.WaitGroup

	// per-entity leader channels and electors
	type entState struct {
		ent      config.EntityConfig
		leaderCh chan bool
		elector  *election.Elector
		isLeader bool
	}

	states := make([]*entState, 0, len(cfg.Entities))

	// start electors
	for _, ent := range cfg.Entities {
		lease := cfg.Global.Election.LeaseDuration
		hb := cfg.Global.Election.HeartbeatInterval
		leaderCh := make(chan bool, 1)
		e := election.NewElector(ent, p, myID, lease, hb)
		states = append(states, &entState{ent: ent, leaderCh: leaderCh, elector: e})
		wg.Add(1)
		go func(es *entState) {
			defer wg.Done()
			es.elector.Run(es.leaderCh)
		}(states[len(states)-1])
	}

	// health loop
	ticker := time.NewTicker(cfg.Global.Health.CheckInterval)
	defer ticker.Stop()

	// map to store last healthy IPs per entity
	lastHealthy := make(map[string][]string)

loop:
	for {
		select {
		case <-stop:
			log.Println("Coordinator stopping...")
			// stop electors
			for _, s := range states {
				s.elector.Stop()
			}
			break loop

		case <-ticker.C:
			// poll for leader statuses (non-blocking)
			for _, s := range states {
				select {
				case isLeader := <-s.leaderCh:
					s.isLeader = isLeader
				default:
					// keep previous state
				}
			}

			// run checks per entity
			for _, s := range states {
				ent := s.ent
				healthyIPs := []string{}
				for _, srv := range ent.Servers {
					alive := health.Check(srv.Address, srv.Check, ent.Health)
					if alive {
						healthyIPs = append(healthyIPs, srv.Address)
					}
				}

				// if leader -> update A records (simple: use first healthy IP)
				if s.isLeader {
					if len(healthyIPs) == 0 {
						log.Printf("[%s] leader but no healthy servers; skipping A update", ent.Name)
						continue
					}

					ip := healthyIPs[0]

					// compare with last state
					prev := lastHealthy[ent.Name]
					if len(prev) > 0 && prev[0] == ip {
						continue
					}

					log.Printf("[%s] leader updating A to %s", ent.Name, ip)
					if err := p.UpdateRecord(ent.Hostname, ip, ent.DNS.Proxied, ent.DNS.TTL); err != nil {
						log.Printf("[%s] failed to update A record: %v", ent.Name, err)
					} else {
						lastHealthy[ent.Name] = []string{ip}
					}
				} else {
					// not leader: optionally record list for diagnostics
					lastHealthy[ent.Name] = healthyIPs
				}
			}
		}
	}

	// wait for all electors to finish cleanly
	wg.Wait()
	log.Println("Coordinator stopped.")
}
