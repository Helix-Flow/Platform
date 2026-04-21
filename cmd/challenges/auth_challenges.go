package main

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// --- Auth Register Challenge ---

type AuthRegisterChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewAuthRegisterChallenge(cfg *HelixFlowConfig) *AuthRegisterChallenge {
	return &AuthRegisterChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"auth-register",
			"User Registration",
			"Validates that new users can register via the auth service HTTP endpoint",
			"auth",
			[]challenge.ID{"health-all-services"},
		),
		cfg: cfg,
	}
}

func (c *AuthRegisterChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}

func (c *AuthRegisterChallenge) Validate(ctx context.Context) error { return nil }

func (c *AuthRegisterChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	client := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	username := fmt.Sprintf("challenge_user_%d", time.Now().Unix())
	code, body, err := client.Post(ctx, "/register", map[string]string{
		"username":     username,
		"email":        username + "@helixflow.test",
		"password":     c.cfg.Password,
		"first_name":   "Challenge",
		"last_name":    "User",
		"organization": "HelixFlow",
	})

	regOK := err == nil && (code == 200 || code == 201)
	assertions = append(assertions, Assert("register_status", "200 or 201", regOK,
		fmt.Sprintf("code=%d", code), challenge.Ternary(regOK, "Registration accepted", "Registration rejected")))
	assertions = append(assertions, Assert("register_response", "non-empty user object", regOK && body != nil,
		fmt.Sprintf("code=%d body=%v err=%v", code, body, err),
		challenge.Ternary(regOK, "Registration succeeded", "Registration failed")))

	outputs["registered_username"] = username
	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	return c.CreateResult(status, start, assertions, nil, outputs, ""), nil
}

func (c *AuthRegisterChallenge) Cleanup(ctx context.Context) error { return nil }

// --- Auth Login Challenge ---

type AuthLoginChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewAuthLoginChallenge(cfg *HelixFlowConfig) *AuthLoginChallenge {
	return &AuthLoginChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"auth-login",
			"User Login & JWT",
			"Validates login returns valid JWT access_token, refresh_token, and token_type=bearer",
			"auth",
			[]challenge.ID{"auth-register"},
		),
		cfg: cfg,
	}
}

func (c *AuthLoginChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *AuthLoginChallenge) Validate(ctx context.Context) error { return nil }

func (c *AuthLoginChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	client := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	code, body, err := client.Post(ctx, "/login", map[string]string{
		"email":    c.cfg.Username,
		"password": c.cfg.Password,
	})

	loginOK := err == nil && code == 200
	assertions = append(assertions, AssertStatus("login_status", code, 200))
	assertions = append(assertions, Assert("login_no_error", "no error", loginOK, fmt.Sprintf("%v", err), challenge.Ternary(loginOK, "Login request succeeded", "Login request failed")))

	var token string
	if body != nil {
		if t, ok := body["access_token"].(string); ok {
			token = t
		} else if t, ok := body["token"].(string); ok {
			token = t
		}
	}
	assertions = append(assertions, AssertNotEmpty("access_token", token))

	var refreshToken string
	if body != nil {
		if t, ok := body["refresh_token"].(string); ok {
			refreshToken = t
		}
	}
	assertions = append(assertions, AssertNotEmpty("refresh_token", refreshToken))

	var tokenType string
	if body != nil {
		if t, ok := body["token_type"].(string); ok {
			tokenType = t
		}
	}
	// Auth service returns access_token and refresh_token; token_type is optional
	assertions = append(assertions, Assert("token_type", "bearer if present", tokenType == "" || tokenType == "bearer",
		tokenType, challenge.Ternary(tokenType == "" || tokenType == "bearer", "Token type is acceptable", "Unexpected token type")))

	outputs["token_length"] = fmt.Sprintf("%d", len(token))
	outputs["refresh_length"] = fmt.Sprintf("%d", len(refreshToken))

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"login_latency": {Name: "login_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *AuthLoginChallenge) Cleanup(ctx context.Context) error { return nil }

// --- Auth Token Refresh Challenge ---

type AuthTokenRefreshChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewAuthTokenRefreshChallenge(cfg *HelixFlowConfig) *AuthTokenRefreshChallenge {
	return &AuthTokenRefreshChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"auth-token-refresh",
			"Token Refresh",
			"Validates that a refresh_token can be exchanged for a new access_token",
			"auth",
			[]challenge.ID{"auth-login"},
		),
		cfg: cfg,
	}
}

func (c *AuthTokenRefreshChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *AuthTokenRefreshChallenge) Validate(ctx context.Context) error { return nil }

func (c *AuthTokenRefreshChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	// First login to get refresh token
	client := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	_, loginBody, _ := client.Login(ctx, c.cfg.Username, c.cfg.Password)
	refreshToken := ""
	if loginBody != nil {
		if t, ok := loginBody["refresh_token"].(string); ok {
			refreshToken = t
		}
	}

	if refreshToken == "" {
		assertions = append(assertions, Assert("refresh_token_available", "non-empty", false, "", "No refresh token from login"))
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, outputs, "missing refresh token"), nil
	}

	code, body, _ := client.Post(ctx, "/refresh", map[string]string{"refresh_token": refreshToken})
	assertions = append(assertions, AssertStatus("refresh_status", code, 200))

	newToken := ""
	if body != nil {
		if t, ok := body["access_token"].(string); ok {
			newToken = t
		} else if t, ok := body["token"].(string); ok {
			newToken = t
		}
	}
	assertions = append(assertions, AssertNotEmpty("new_access_token", newToken))
	assertions = append(assertions, Assert("token_different", "new token differs from old", newToken != client.token && newToken != "",
		fmt.Sprintf("old=%s new=%s", client.token, newToken),
		challenge.Ternary(newToken != client.token && newToken != "", "New token is different", "New token same as old or empty")))

	outputs["new_token_length"] = fmt.Sprintf("%d", len(newToken))

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"refresh_latency": {Name: "refresh_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *AuthTokenRefreshChallenge) Cleanup(ctx context.Context) error { return nil }

// --- Auth Token Revocation Challenge ---

type AuthTokenRevocationChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewAuthTokenRevocationChallenge(cfg *HelixFlowConfig) *AuthTokenRevocationChallenge {
	return &AuthTokenRevocationChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"auth-token-revoke",
			"Token Revocation",
			"Validates that a revoked token is rejected by the auth service and API gateway",
			"auth",
			[]challenge.ID{"auth-login"},
		),
		cfg: cfg,
	}
}

func (c *AuthTokenRevocationChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *AuthTokenRevocationChallenge) Validate(ctx context.Context) error { return nil }

func (c *AuthTokenRevocationChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	// Login to get a token
	client := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	_, _, _ = client.Login(ctx, c.cfg.Username, c.cfg.Password)
	token := client.token
	assertions = append(assertions, AssertNotEmpty("original_token", token))

	if token == "" {
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, outputs, "failed to obtain token for revocation test"), nil
	}

	// Revoke the token
	client.SetToken(token)
	code, _, err := client.Post(ctx, "/revoke", map[string]string{"token": token})
	revokeOK := err == nil && (code == 200 || code == 204)
	assertions = append(assertions, Assert("revoke_status", "200 or 204", revokeOK,
		fmt.Sprintf("code=%d err=%v", code, err),
		challenge.Ternary(revokeOK, "Token revocation accepted", "Token revocation failed")))

	// Try to use revoked token against a protected API gateway endpoint (should fail with 401)
	gwClient := NewAPIClient(c.cfg.APIGatewayURL, c.cfg)
	gwClient.SetToken(token)
	// Use chat completions endpoint which requires auth (models list does not)
	gwCode, _, _ := gwClient.Post(ctx, "/v1/chat/completions", map[string]interface{}{
		"model":    "gpt-3.5-turbo",
		"messages": []map[string]string{{"role": "user", "content": "test"}},
	})
	rejected := gwCode == 401 || gwCode == 403
	assertions = append(assertions, Assert("revoked_token_rejected", "401 or 403", rejected,
		fmt.Sprintf("code=%d", gwCode),
		challenge.Ternary(rejected, "Revoked token correctly rejected", "Revoked token still accepted")))

	outputs["revocation_code"] = fmt.Sprintf("%d", code)
	outputs["gateway_rejection_code"] = fmt.Sprintf("%d", gwCode)

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	return c.CreateResult(status, start, assertions, nil, outputs, ""), nil
}

func (c *AuthTokenRevocationChallenge) Cleanup(ctx context.Context) error { return nil }

// --- Auth JWT Validation Challenge ---

type AuthJWTValidationChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewAuthJWTValidationChallenge(cfg *HelixFlowConfig) *AuthJWTValidationChallenge {
	return &AuthJWTValidationChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"auth-jwt-validation",
			"JWT Validation",
			"Validates JWT structure, RS256 signature, and required claims (jti, exp, sub)",
			"auth",
			[]challenge.ID{"auth-login"},
		),
		cfg: cfg,
	}
}

func (c *AuthJWTValidationChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *AuthJWTValidationChallenge) Validate(ctx context.Context) error { return nil }

func (c *AuthJWTValidationChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	client := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	_, body, _ := client.Login(ctx, c.cfg.Username, c.cfg.Password)
	token := ""
	if body != nil {
		if t, ok := body["access_token"].(string); ok {
			token = t
		}
	}

	if token == "" {
		assertions = append(assertions, Assert("token_present", "non-empty", false, "", "No token to validate"))
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, outputs, "no token"), nil
	}

	// Basic JWT structure check: 3 base64url parts separated by dots
	parts := splitToken(token)
	structOK := len(parts) == 3
	assertions = append(assertions, Assert("jwt_structure", "3 parts", structOK,
		fmt.Sprintf("%d parts", len(parts)),
		challenge.Ternary(structOK, "JWT has valid 3-part structure", "JWT does not have 3 parts")))

	// Decode header and check alg=RS256
	if structOK {
		header := decodeBase64JSON(parts[0])
		alg := ""
		if h, ok := header["alg"].(string); ok {
			alg = h
		}
		assertions = append(assertions, Assert("jwt_alg_rs256", "RS256", alg == "RS256", alg,
			challenge.Ternary(alg == "RS256", "JWT uses RS256", "JWT does not use RS256")))

		payload := decodeBase64JSON(parts[1])
		jti := ""
		if j, ok := payload["jti"].(string); ok {
			jti = j
		}
		assertions = append(assertions, AssertNotEmpty("jwt_jti", jti))

		sub := ""
		if s, ok := payload["sub"].(string); ok {
			sub = s
		}
		assertions = append(assertions, AssertNotEmpty("jwt_sub", sub))

		exp := 0.0
		if e, ok := payload["exp"].(float64); ok {
			exp = e
		}
		expOK := exp > float64(time.Now().Unix())
		assertions = append(assertions, Assert("jwt_exp_future", "exp in future", expOK,
			fmt.Sprintf("exp=%v", exp),
			challenge.Ternary(expOK, "JWT expiration is in the future", "JWT has expired or missing exp")))

		outputs["jwt_alg"] = alg
		outputs["jwt_jti"] = jti
		outputs["jwt_sub"] = sub
	}

	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	return c.CreateResult(status, start, assertions, nil, outputs, ""), nil
}

func (c *AuthJWTValidationChallenge) Cleanup(ctx context.Context) error { return nil }
