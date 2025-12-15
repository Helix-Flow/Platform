package main_test

import (
	"encoding/json"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
	uuid "github.com/google/uuid"
)

func TestTokenGenerationWithJTI(t *testing.T) {
	resp, err := http.Post("http://localhost:8082/auth/login", "application/json", strings.NewReader(`{"username":"test", "password":"testpass"}`))
	if err != nil {
		t.Fatalf("Failed to make login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Login request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var responseMap map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	accessToken, hasAccess := responseMap["access_token"]
	refreshToken, hasRefresh := responseMap["refresh_token"]

	if !hasAccess || !hasRefresh {
		t.Fatalf("Expected both tokens in response, got: %v", responseMap)
	}

	verifyJTI(t, accessToken, "access")
	verifyJTI(t, refreshToken, "refresh")
}

func verifyJTI(t *testing.T, token string, tokenType string) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("Invalid %s token format", tokenType)
	}

	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]interface{}
	json.Unmarshal(payload, &claims)

	jti, hasJTI := claims["jti"]
	if !hasJTI {
		t.Errorf("%s token missing JTI claim", tokenType)
		return
	}

	parsedUUID, err := uuid.Parse(jti.(string))
	if err != nil {
		t.Errorf("%s token JTI is not a valid UUID: %v", tokenType, jti)
		return
	}

	if parsedUUID.Version() != 4 {
		t.Errorf("%s token JTI must be UUID version 4, got version %d", tokenType, parsedUUID.Version())
	}
}