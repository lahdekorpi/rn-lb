package health

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"rn-lb/internal/config"
)

type Result struct {
	ServerID string
	Healthy  bool
	Error    error
}

func CheckServer(server config.ServerEntry) Result {
	c := server.Check

	timeout := c.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	switch c.Type {
	case "http":
		return checkHTTP(server, timeout)
	case "tcp":
		return checkTCP(server, timeout)
	default:
		return Result{ServerID: server.ID, Healthy: false, Error: fmt.Errorf("unsupported check type: %s", c.Type)}
	}
}

func checkHTTP(server config.ServerEntry, timeout time.Duration) Result {
	c := server.Check

	url := fmt.Sprintf("%s://%s:%d%s",
		firstNonEmpty(c.Protocol, "http"),
		server.Address,
		c.Port,
		c.Path)

	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest(firstNonEmpty(c.Method, "GET"), url, nil)
	if err != nil {
		return Result{ServerID: server.ID, Healthy: false, Error: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{ServerID: server.ID, Healthy: false, Error: err}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// if valid_status is empty, accept 200..399
	if len(c.ValidStatus) == 0 {
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return Result{ServerID: server.ID, Healthy: true}
		}
		return Result{ServerID: server.ID, Healthy: false, Error: fmt.Errorf("status %d", resp.StatusCode)}
	}

	for _, code := range c.ValidStatus {
		if resp.StatusCode == code {
			return Result{ServerID: server.ID, Healthy: true}
		}
	}

	return Result{ServerID: server.ID, Healthy: false, Error: fmt.Errorf("invalid status %d", resp.StatusCode)}
}

func checkTCP(server config.ServerEntry, timeout time.Duration) Result {
	c := server.Check

	addr := fmt.Sprintf("%s:%d", server.Address, c.Port)

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return Result{ServerID: server.ID, Healthy: false, Error: err}
	}
	conn.Close()
	return Result{ServerID: server.ID, Healthy: true}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
