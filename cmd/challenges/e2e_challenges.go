package main

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// --- E2E Full Chat Flow Challenge ---

type E2EFullChatFlowChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewE2EFullChatFlowChallenge(cfg *HelixFlowConfig) *E2EFullChatFlowChallenge {
	return &E2EFullChatFlowChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"e2e-chat-flow",
			"End-to-End Chat Flow",
			"Validates the complete flow: auth -> gateway -> inference -> response, with token validation at each step",
			"e2e",
			[]challenge.ID{"auth-login", "gateway-chat", "inference-model-exec"},
		),
		cfg: cfg,
	}
}

func (c *E2EFullChatFlowChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *E2EFullChatFlowChallenge) Validate(ctx context.Context) error { return nil }

func (c *E2EFullChatFlowChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	// Step 1: Login
	authClient := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	code, _, err := authClient.Login(ctx, c.cfg.Username, c.cfg.Password)
	loginOK := err == nil && code == 200
	assertions = append(assertions, Assert("e2e_login", "HTTP 200", loginOK,
		fmt.Sprintf("code=%d", code), challenge.Ternary(loginOK, "Login succeeded", "Login failed")))
	if !loginOK {
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, outputs, "login failed"), nil
	}

	token := authClient.token
	assertions = append(assertions, AssertNotEmpty("e2e_token", token))

	// Step 2: List models via gateway
	gwClient := NewAPIClient(c.cfg.APIGatewayURL, c.cfg)
	gwClient.SetToken(token)
	code, modelsBody, err := gwClient.Get(ctx, "/v1/models")
	modelsOK := err == nil && code == 200
	assertions = append(assertions, Assert("e2e_models", "HTTP 200", modelsOK,
		fmt.Sprintf("code=%d", code), challenge.Ternary(modelsOK, "Models list succeeded", "Models list failed")))

	var modelID string
	if modelsBody != nil {
		if data, ok := modelsBody["data"].([]interface{}); ok && len(data) > 0 {
			if first, ok := data[0].(map[string]interface{}); ok {
				if id, ok := first["id"].(string); ok {
					modelID = id
				}
			}
		}
	}
	assertions = append(assertions, AssertNotEmpty("e2e_model_id", modelID))

	// Step 3: Chat completion
	code, chatBody, err := gwClient.Post(ctx, "/v1/chat/completions", map[string]interface{}{
		"model":    modelID,
		"messages": []map[string]string{{"role": "user", "content": "Hello from E2E challenge"}},
	})
	chatOK := err == nil && code == 200
	assertions = append(assertions, Assert("e2e_chat", "HTTP 200", chatOK,
		fmt.Sprintf("code=%d", code), challenge.Ternary(chatOK, "Chat completion succeeded", "Chat completion failed")))

	var content string
	if chatBody != nil {
		if choices, ok := chatBody["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					if c, ok := msg["content"].(string); ok {
						content = c
					}
				}
			}
		}
	}
	assertions = append(assertions, AssertNotEmpty("e2e_chat_content", content))

	// Step 4: Verify token still works after inference
	code, _, _ = gwClient.Get(ctx, "/v1/models")
	tokenStillValid := code == 200
	assertions = append(assertions, Assert("e2e_token_still_valid", "HTTP 200", tokenStillValid,
		fmt.Sprintf("code=%d", code), challenge.Ternary(tokenStillValid, "Token still valid after inference", "Token invalidated unexpectedly")))

	outputs["model_used"] = modelID
	outputs["response_preview"] = content

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"e2e_latency": {Name: "e2e_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *E2EFullChatFlowChallenge) Cleanup(ctx context.Context) error { return nil }

// --- E2E Token Lifecycle Challenge ---

type E2ETokenLifecycleChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewE2ETokenLifecycleChallenge(cfg *HelixFlowConfig) *E2ETokenLifecycleChallenge {
	return &E2ETokenLifecycleChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"e2e-token-lifecycle",
			"Token Lifecycle",
			"Validates complete token lifecycle: login -> use -> refresh -> use new -> revoke -> reject revoked",
			"e2e",
			[]challenge.ID{"auth-login", "auth-token-refresh", "auth-token-revoke"},
		),
		cfg: cfg,
	}
}

func (c *E2ETokenLifecycleChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *E2ETokenLifecycleChallenge) Validate(ctx context.Context) error { return nil }

func (c *E2ETokenLifecycleChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	authClient := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)

	// 1. Login
	_, loginBody, _ := authClient.Login(ctx, c.cfg.Username, c.cfg.Password)
	originalToken := authClient.token
	assertions = append(assertions, AssertNotEmpty("lifecycle_original_token", originalToken))

	// 2. Use token against gateway
	gwClient := NewAPIClient(c.cfg.APIGatewayURL, c.cfg)
	gwClient.SetToken(originalToken)
	code, _, _ := gwClient.Get(ctx, "/v1/models")
	assertions = append(assertions, AssertStatus("lifecycle_use_original", code, 200))

	// 3. Refresh token
	refreshToken := ""
	if loginBody != nil {
		if t, ok := loginBody["refresh_token"].(string); ok {
			refreshToken = t
		}
	}
	if refreshToken == "" {
		assertions = append(assertions, Assert("lifecycle_refresh_token", "present", false, "", "No refresh token"))
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, outputs, "no refresh token"), nil
	}

	code, refreshBody, _ := authClient.Post(ctx, "/refresh", map[string]string{"refresh_token": refreshToken})
	assertions = append(assertions, AssertStatus("lifecycle_refresh_status", code, 200))

	newToken := ""
	if refreshBody != nil {
		if t, ok := refreshBody["access_token"].(string); ok {
			newToken = t
		} else if t, ok := refreshBody["token"].(string); ok {
			newToken = t
		}
	}
	assertions = append(assertions, AssertNotEmpty("lifecycle_new_token", newToken))
	assertions = append(assertions, Assert("lifecycle_token_changed", "different from original", newToken != originalToken && newToken != "",
		fmt.Sprintf("old=%s new=%s", originalToken, newToken), challenge.Ternary(newToken != originalToken, "Token changed after refresh", "Token did not change")))

	// 4. Use new token
	gwClient.SetToken(newToken)
	code, _, _ = gwClient.Get(ctx, "/v1/models")
	assertions = append(assertions, AssertStatus("lifecycle_use_new", code, 200))

	// 5. Revoke new token via auth service
	authClient.SetToken(newToken)
	code, _, _ = authClient.Post(ctx, "/revoke", map[string]string{"token": newToken})
	revokeOK := code == 200 || code == 204
	assertions = append(assertions, Assert("lifecycle_revoke", "200 or 204", revokeOK,
		fmt.Sprintf("code=%d", code), challenge.Ternary(revokeOK, "Revocation accepted", "Revocation failed")))

	// 6. Verify revoked token is rejected on protected endpoint
	gwClient.SetToken(newToken)
	code, _, _ = gwClient.Post(ctx, "/v1/chat/completions", map[string]interface{}{
		"model":    "gpt-3.5-turbo",
		"messages": []map[string]string{{"role": "user", "content": "test"}},
	})
	rejected := code == 401 || code == 403
	assertions = append(assertions, Assert("lifecycle_revoked_rejected", "401 or 403", rejected,
		fmt.Sprintf("code=%d", code), challenge.Ternary(rejected, "Revoked token rejected", "Revoked token still accepted")))

	outputs["original_token_prefix"] = originalToken[:min(10, len(originalToken))]
	outputs["new_token_prefix"] = newToken[:min(10, len(newToken))]

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"lifecycle_latency": {Name: "lifecycle_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *E2ETokenLifecycleChallenge) Cleanup(ctx context.Context) error { return nil }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
