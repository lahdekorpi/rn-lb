package health

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"rn-lb/internal/config"
)

// Check suorittaa health checkin serverille configin perusteella.
// Palauttaa true jos serveri vastaa terveenä.
func Check(addr string, hc config.CheckEntry, he config.HealthConfig) bool {
	timeout := he.Timeout
	if hc.Timeout > 0 {
		timeout = hc.Timeout
	}

	retries := he.Retries
	retryWait := he.RetryWait
	if retries <= 0 {
		retries = 1
	}

	for i := 0; i < retries; i++ {
		var ok bool
		var err error

		switch hc.Type {
		case "http":
			ok, err = httpCheck(addr, hc, timeout)
		case "tcp":
			ok, err = tcpCheck(addr, hc, timeout)
		default:
			log.Printf("Unknown health check type: %s", hc.Type)
			return false
		}

		if ok {
			return true
		}

		log.Printf("Health check failed (%s) attempt %d/%d: %v",
			hc.Type, i+1, retries, err)
		time.Sleep(retryWait)
	}
	return false
}

func httpCheck(addr string, hc config.CheckEntry, timeout time.Duration) (bool, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	// Käytetään osoitetta, johon lisätään portti ja path
	protocol := hc.Protocol
	if protocol == "" {
		protocol = "http"
	}

	url := fmt.Sprintf("%s://%s:%d%s", protocol, addr, hc.Port, hc.Path)
	if hc.Method == "" {
		hc.Method = "GET"
	}

	req, err := http.NewRequest(hc.Method, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// Jos valid_status lista määritelty, käytetään sitä
	if len(hc.ValidStatus) > 0 {
		for _, code := range hc.ValidStatus {
			if resp.StatusCode == code {
				return true, nil
			}
		}
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	// muuten hyväksytään 200–399
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, nil
	}
	return false, fmt.Errorf("bad status %d", resp.StatusCode)
}

func tcpCheck(addr string, hc config.CheckEntry, timeout time.Duration) (bool, error) {
	target := fmt.Sprintf("%s:%d", addr, hc.Port)
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}
