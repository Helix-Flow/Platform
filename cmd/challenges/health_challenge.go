package main

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// HealthCheckChallenge validates all core services are reachable and healthy.
type HealthCheckChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

// NewHealthCheckChallenge creates the health check challenge.
func NewHealthCheckChallenge(cfg *HelixFlowConfig) *HealthCheckChallenge {
	return &HealthCheckChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"health-all-services",
			"All Services Health Check",
			"Validates API Gateway, Auth Service, Monitoring, and Inference Pool are reachable and report healthy status",
			"health",
			nil,
		),
		cfg: cfg,
	}
}

// Configure applies runtime configuration.
func (c *HealthCheckChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}

// Validate checks preconditions.
func (c *HealthCheckChallenge) Validate(ctx context.Context) error {
	return nil
}

// Execute runs health checks against all services.
func (c *HealthCheckChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	client := NewAPIClient(c.cfg.APIGatewayURL, c.cfg)

	// API Gateway health
	code, body, err := client.Get(ctx, "/health")
	gwStatus := ""
	if body != nil {
		if s, ok := body["status"].(string); ok {
			gwStatus = s
		}
	}
	gwOK := err == nil && code == 200 && gwStatus == "healthy"
	assertions = append(assertions, Assert("api_gateway_health", "HTTP 200 status=healthy", gwOK,
		fmt.Sprintf("code=%d status=%q err=%v", code, gwStatus, err),
		challenge.Ternary(gwOK, "API Gateway is healthy", "API Gateway health check failed")))
	outputs["api_gateway_status"] = gwStatus

	// Auth Service health
	authClient := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	code, body, err = authClient.Get(ctx, "/health")
	authStatus := ""
	if body != nil {
		if s, ok := body["status"].(string); ok {
			authStatus = s
		}
	}
	authOK := err == nil && code == 200 && authStatus == "healthy"
	assertions = append(assertions, Assert("auth_service_health", "HTTP 200 status=healthy", authOK,
		fmt.Sprintf("code=%d status=%q err=%v", code, authStatus, err),
		challenge.Ternary(authOK, "Auth Service is healthy", "Auth Service health check failed")))
	outputs["auth_service_status"] = authStatus

	// Monitoring health
	monClient := NewAPIClient(c.cfg.GetMonitoringURL(), c.cfg)
	code, body, err = monClient.Get(ctx, "/health")
	monStatus := ""
	if body != nil {
		if s, ok := body["status"].(string); ok {
			monStatus = s
		}
	}
	monOK := err == nil && code == 200 && monStatus == "healthy"
	assertions = append(assertions, Assert("monitoring_health", "HTTP 200 status=healthy", monOK,
		fmt.Sprintf("code=%d status=%q err=%v", code, monStatus, err),
		challenge.Ternary(monOK, "Monitoring is healthy", "Monitoring health check failed")))
	outputs["monitoring_status"] = monStatus

	// Inference Pool TCP connectivity
	tcpOK := checkTCP(c.cfg.InferencePoolURL) == nil
	assertions = append(assertions, Assert("inference_pool_tcp", "TCP connectable", tcpOK,
		challenge.Ternary(tcpOK, "connected", "connection refused"),
		challenge.Ternary(tcpOK, "Inference Pool TCP port is open", "Inference Pool TCP port unreachable")))

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
			break
		}
	}

	metrics := map[string]challenge.MetricValue{
		"health_check_latency": {Name: "health_check_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}

	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

// Cleanup releases resources.
func (c *HealthCheckChallenge) Cleanup(ctx context.Context) error {
	return nil
}
