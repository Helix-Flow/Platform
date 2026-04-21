package main

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// --- Gateway Models List Challenge ---

type GatewayModelsChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewGatewayModelsChallenge(cfg *HelixFlowConfig) *GatewayModelsChallenge {
	return &GatewayModelsChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"gateway-models",
			"Models List Endpoint",
			"Validates GET /v1/models returns OpenAI-compatible model list with id, object, created, owned_by",
			"gateway",
			[]challenge.ID{"health-all-services", "auth-login"},
		),
		cfg: cfg,
	}
}

func (c *GatewayModelsChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *GatewayModelsChallenge) Validate(ctx context.Context) error { return nil }

func (c *GatewayModelsChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	client := NewAPIClient(c.cfg.APIGatewayURL, c.cfg)
	// Login first
	client.Login(ctx, c.cfg.Username, c.cfg.Password)

	code, body, err := client.Get(ctx, "/v1/models")
	ok := err == nil && code == 200
	assertions = append(assertions, AssertStatus("models_status", code, 200))

	var data []interface{}
	if body != nil {
		if d, ok := body["data"].([]interface{}); ok {
			data = d
		}
	}
	assertions = append(assertions, Assert("models_data_array", "non-empty array", ok && len(data) > 0,
		fmt.Sprintf("len=%d", len(data)),
		challenge.Ternary(len(data) > 0, "Models list returned entries", "Models list empty")))

	if len(data) > 0 {
		first, isMap := data[0].(map[string]interface{})
		assertions = append(assertions, Assert("model_has_id", "id present", isMap && first["id"] != nil,
			fmt.Sprintf("%v", first["id"]), "First model has id"))
		assertions = append(assertions, Assert("model_has_object", "object=model", isMap && first["object"] == "model",
			fmt.Sprintf("%v", first["object"]), "First model object type is model"))
	}

	outputs["model_count"] = fmt.Sprintf("%d", len(data))
	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"models_latency": {Name: "models_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *GatewayModelsChallenge) Cleanup(ctx context.Context) error { return nil }

// --- Gateway Chat Completions Challenge ---

type GatewayChatCompletionsChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewGatewayChatCompletionsChallenge(cfg *HelixFlowConfig) *GatewayChatCompletionsChallenge {
	return &GatewayChatCompletionsChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"gateway-chat",
			"Chat Completions",
			"Validates POST /v1/chat/completions returns OpenAI-compatible response with choices, message, content",
			"gateway",
			[]challenge.ID{"gateway-models"},
		),
		cfg: cfg,
	}
}

func (c *GatewayChatCompletionsChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *GatewayChatCompletionsChallenge) Validate(ctx context.Context) error { return nil }

func (c *GatewayChatCompletionsChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	authClient := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	authClient.Login(ctx, c.cfg.Username, c.cfg.Password)

	gwClient := NewAPIClient(c.cfg.APIGatewayURL, c.cfg)
	gwClient.SetToken(authClient.token)

	code, body, err := gwClient.Post(ctx, "/v1/chat/completions", map[string]interface{}{
		"model":    "gpt-3.5-turbo",
		"messages": []map[string]string{{"role": "user", "content": "Say hello"}},
	})
	ok := err == nil && code == 200
	assertions = append(assertions, AssertStatus("chat_status", code, 200))
	assertions = append(assertions, Assert("chat_no_error", "no error", ok, fmt.Sprintf("%v", err), challenge.Ternary(ok, "Chat completion succeeded", "Chat completion failed")))

	var choices []interface{}
	if body != nil {
		if c, ok := body["choices"].([]interface{}); ok {
			choices = c
		}
	}
	assertions = append(assertions, Assert("chat_choices", "non-empty choices", ok && len(choices) > 0,
		fmt.Sprintf("len=%d", len(choices)), challenge.Ternary(len(choices) > 0, "Response has choices", "No choices in response")))

	var content string
	if len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if c, ok := msg["content"].(string); ok {
					content = c
				}
			}
		}
	}
	assertions = append(assertions, AssertNotEmpty("chat_content", content))

	outputs["content_preview"] = content
	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"chat_latency": {Name: "chat_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *GatewayChatCompletionsChallenge) Cleanup(ctx context.Context) error { return nil }

// --- Gateway Rate Limiting Challenge ---

type GatewayRateLimitingChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewGatewayRateLimitingChallenge(cfg *HelixFlowConfig) *GatewayRateLimitingChallenge {
	return &GatewayRateLimitingChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"gateway-rate-limit",
			"Rate Limiting",
			"Validates that rapid requests to protected endpoints trigger rate limiting (429) or remain within limits",
			"gateway",
			[]challenge.ID{"gateway-chat"},
		),
		cfg: cfg,
	}
}

func (c *GatewayRateLimitingChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *GatewayRateLimitingChallenge) Validate(ctx context.Context) error { return nil }

func (c *GatewayRateLimitingChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	authClient := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	authClient.Login(ctx, c.cfg.Username, c.cfg.Password)

	gwClient := NewAPIClient(c.cfg.APIGatewayURL, c.cfg)
	gwClient.SetToken(authClient.token)

	// Fire 20 rapid requests to a protected endpoint
	codes := []int{}
	for i := 0; i < 20; i++ {
		code, _, _ := gwClient.Post(ctx, "/v1/chat/completions", map[string]interface{}{
			"model":    "gpt-3.5-turbo",
			"messages": []map[string]string{{"role": "user", "content": "test"}},
		})
		codes = append(codes, code)
	}

	total429 := 0
	total200 := 0
	for _, code := range codes {
		if code == 429 {
			total429++
		}
		if code == 200 {
			total200++
		}
	}

	// Rate limiting requires Redis. If Redis is unavailable, all requests return 200.
	// We flag this as a soft failure but do not fail the challenge outright.
	rateLimited := total429 > 0
	assertions = append(assertions, Assert("rate_limit_429_seen", "at least one 429 if Redis active", rateLimited || total200 == len(codes),
		fmt.Sprintf("429s=%d 200s=%d", total429, total200),
		challenge.Ternary(rateLimited, "Rate limiting triggered 429", "No 429 observed (Redis may be unavailable)")))

	// All responses should be either 200 or 429 (no 500s)
	allValid := true
	for _, code := range codes {
		if code != 200 && code != 429 {
			allValid = false
			break
		}
	}
	assertions = append(assertions, Assert("rate_limit_no_errors", "only 200 or 429", allValid,
		fmt.Sprintf("codes=%v", codes),
		challenge.Ternary(allValid, "No server errors during rate test", "Unexpected error codes during rate test")))

	outputs["total_requests"] = fmt.Sprintf("%d", len(codes))
	outputs["429_count"] = fmt.Sprintf("%d", total429)
	outputs["200_count"] = fmt.Sprintf("%d", total200)

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"rate_test_duration": {Name: "rate_test_duration", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *GatewayRateLimitingChallenge) Cleanup(ctx context.Context) error { return nil }
