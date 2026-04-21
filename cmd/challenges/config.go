package main

import (
	"fmt"
	"os"
)

// HelixFlowConfig holds service endpoints and credentials for challenges.
type HelixFlowConfig struct {
	APIGatewayURL     string
	AuthServiceURL    string
	AuthGRPCURL       string
	InferencePoolURL  string
	MonitoringURL     string
	Username          string
	Password          string
	AdminUsername     string
	AdminPassword     string
	TLSCertPath       string
	TLSKeyPath        string
	TLSCAPath         string
	SkipTLSVerify     bool
	ResultsDir        string
	LogsDir           string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *HelixFlowConfig {
	return &HelixFlowConfig{
		APIGatewayURL:    getEnv("API_GATEWAY_URL", "https://localhost:8443"),
		AuthServiceURL:   getEnv("AUTH_SERVICE_URL", "http://localhost:8082"),
		AuthGRPCURL:      getEnv("AUTH_SERVICE_GRPC", "localhost:8081"),
		InferencePoolURL: getEnv("INFERENCE_POOL_URL", "localhost:50051"),
		MonitoringURL:    getEnv("MONITORING_URL", "http://localhost:8083"),
		Username:         getEnv("TEST_USERNAME", "testuser"),
		Password:         getEnv("TEST_PASSWORD", "password"),
		AdminUsername:    getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "admin123"),
		TLSCertPath:      getEnv("TLS_CERT", "../../certs/api-gateway.crt"),
		TLSKeyPath:       getEnv("TLS_KEY", "../../certs/api-gateway-key.pem"),
		TLSCAPath:        getEnv("TLS_CA", "../../certs/ca.pem"),
		SkipTLSVerify:    getEnv("SKIP_TLS_VERIFY", "true") == "true",
		ResultsDir:       getEnv("RESULTS_DIR", "results"),
		LogsDir:          getEnv("LOGS_DIR", "logs"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// BaseURL returns the API gateway base URL.
func (c *HelixFlowConfig) BaseURL() string {
	return c.APIGatewayURL
}

// AuthURL returns the auth service HTTP base URL.
func (c *HelixFlowConfig) AuthURL() string {
	return c.AuthServiceURL
}

// GetMonitoringURL returns the monitoring service base URL.
func (c *HelixFlowConfig) GetMonitoringURL() string {
	return c.MonitoringURL
}

// String returns a redacted config summary.
func (c *HelixFlowConfig) String() string {
	return fmt.Sprintf(
		"HelixFlowConfig{Gateway: %s, Auth: %s, Inference: %s, Monitoring: %s, User: %s}",
		c.APIGatewayURL, c.AuthServiceURL, c.InferencePoolURL, c.MonitoringURL, c.Username,
	)
}
