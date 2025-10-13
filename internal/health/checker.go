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

// Check suorittaa health checkin annetulle palvelimelle.
// Palauttaa true jos palvelin vastaa terveenä.
func Check(serverAddr string, check config.CheckEntry, healthCfg config.HealthConfig) bool {
	timeout := healthCfg.Timeout
	if check.Timeout > 0 {
		timeout = check.Timeout
	}
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	for i := 0; i < 3; i++ { // oletuksena 3 yritystä
		ok, err := performCheck(serverAddr, check, timeout)
		if ok {
			return true
		}
		log.Printf("Health check failed for %s (attempt %d/3): %v", serverAddr, i+1, err)
		time.Sleep(1 * time.Second)
	}
	return false
}

// performCheck valitsee oikean check-tyypin (http tai tcp)
func performCheck(serverAddr string, check config.CheckEntry, timeout time.Duration) (bool, error) {
	switch check.Type {
	case "http":
		return httpCheck(serverAddr, check, timeout)
	case "tcp":
		return tcpCheck(serverAddr, check, timeout)
	default:
		return false, fmt.Errorf("unknown health check type: %s", check.Type)
	}
}

// HTTP health check
func httpCheck(serverAddr string, check config.CheckEntry, timeout time.Duration) (bool, error) {
	client := &http.Client{Timeout: timeout}

	protocol := check.Protocol
	if protocol == "" {
		protocol = "http"
	}

	fullURL := fmt.Sprintf("%s://%s:%d%s", protocol, serverAddr, check.Port, check.Path)

	method := check.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// hyväksy 200–399, ellei muuta määritelty
	if len(check.ValidStatus) > 0 {
		for _, code := range check.ValidStatus {
			if resp.StatusCode == code {
				return true, nil
			}
		}
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, nil
	}
	return false, fmt.Errorf("bad status %d", resp.StatusCode)
}

// TCP health check (IPv4 + IPv6 compatible)
func tcpCheck(serverAddr string, check config.CheckEntry, timeout time.Duration) (bool, error) {
	addr := net.JoinHostPort(serverAddr, fmt.Sprintf("%d", check.Port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}
