package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// preflightHealthChecks verifies all required services are reachable.
func preflightHealthChecks(ctx context.Context, cfg *HelixFlowConfig) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig(cfg),
		},
	}

	checks := []struct {
		name string
		url  string
	}{
		{"API Gateway", cfg.APIGatewayURL + "/health"},
		{"Auth Service", cfg.AuthServiceURL + "/health"},
		{"Monitoring", cfg.MonitoringURL + "/health"},
	}

	for _, c := range checks {
		if err := checkEndpoint(ctx, client, c.name, c.url); err != nil {
			return err
		}
	}

	// Inference pool is gRPC — check TCP connectivity
	if err := checkTCP(cfg.InferencePoolURL); err != nil {
		return fmt.Errorf("inference pool unreachable: %w", err)
	}

	return nil
}

func checkEndpoint(ctx context.Context, client *http.Client, name, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("%s: invalid URL %s: %w", name, endpoint, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %d", name, resp.StatusCode)
	}
	fmt.Printf("  ✓ %s healthy\n", name)
	return nil
}

func tlsConfig(cfg *HelixFlowConfig) *tls.Config {
	if cfg.SkipTLSVerify {
		return &tls.Config{InsecureSkipVerify: true}
	}
	return &tls.Config{}
}

func checkTCP(addr string) error {
	if !strings.Contains(addr, ":") {
		addr = addr + ":50051"
	}
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	fmt.Printf("  ✓ Inference Pool reachable (TCP %s)\n", addr)
	return nil
}
