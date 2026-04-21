package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// APIClient is a lightweight HTTP client for HelixFlow challenges.
type APIClient struct {
	baseURL   string
	token     string
	client    *http.Client
	cfg       *HelixFlowConfig
}

// NewAPIClient creates a client for the given base URL.
func NewAPIClient(baseURL string, cfg *HelixFlowConfig) *APIClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipTLSVerify},
	}
	return &APIClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second, Transport: transport},
		cfg:     cfg,
	}
}

// SetToken sets the Bearer token.
func (c *APIClient) SetToken(token string) {
	c.token = token
}

// Get performs a GET request.
func (c *APIClient) Get(ctx context.Context, path string) (int, map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return 0, nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.do(req)
}

// Post performs a POST request with JSON body.
func (c *APIClient) Post(ctx context.Context, path string, body interface{}) (int, map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.do(req)
}

func (c *APIClient) do(req *http.Request) (int, map[string]interface{}, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		return resp.StatusCode, nil, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return resp.StatusCode, map[string]interface{}{"raw": string(body)}, nil
	}
	return resp.StatusCode, result, nil
}

// Login performs auth service login and sets the token.
func (c *APIClient) Login(ctx context.Context, username, password string) (int, map[string]interface{}, error) {
	code, resp, err := c.Post(ctx, "/login", map[string]string{
		"email":    username,
		"password": password,
	})
	if err != nil {
		return code, resp, err
	}
	if resp != nil {
		if token, ok := resp["access_token"].(string); ok && token != "" {
			c.SetToken(token)
		} else if token, ok := resp["token"].(string); ok && token != "" {
			c.SetToken(token)
		}
	}
	return code, resp, err
}

// Assert adds a structured assertion result.
func Assert(target, expected string, passed bool, actual, msg string) challenge.AssertionResult {
	return challenge.AssertionResult{
		Type:     "not_empty",
		Target:   target,
		Expected: expected,
		Actual:   actual,
		Passed:   passed,
		Message:  msg,
	}
}

// AssertStatus checks HTTP status code.
func AssertStatus(target string, actual, expected int) challenge.AssertionResult {
	passed := actual == expected
	return challenge.AssertionResult{
		Type:     "equals",
		Target:   target,
		Expected: expected,
		Actual:   actual,
		Passed:   passed,
		Message: challenge.Ternary(passed,
			fmt.Sprintf("%s: expected %d, got %d", target, expected, actual),
			fmt.Sprintf("%s: expected %d, got %d", target, expected, actual)),
	}
}

// AssertNotEmpty checks a string is not empty.
func AssertNotEmpty(target, actual string) challenge.AssertionResult {
	passed := actual != ""
	return challenge.AssertionResult{
		Type:     "not_empty",
		Target:   target,
		Expected: "non-empty",
		Actual:   actual,
		Passed:   passed,
		Message:  challenge.Ternary(passed, target+" is not empty", target+" is empty"),
	}
}

// AssertContains checks a string contains a substring.
func AssertContains(target, actual, expected string) challenge.AssertionResult {
	passed := contains(actual, expected)
	return challenge.AssertionResult{
		Type:     "contains",
		Target:   target,
		Expected: expected,
		Actual:   actual,
		Passed:   passed,
		Message:  challenge.Ternary(passed, target+" contains "+expected, target+" does not contain "+expected),
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
