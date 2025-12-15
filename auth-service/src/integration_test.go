package main_test

import (
	"encoding/json"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"github.com/google/uuid"
)

func TestTokenGenerationWithJTI(t *testing.T) {
	resp, err := http.Post("http://localhost:8082/auth/login", "application/json", strings.NewReader(`{"username":"test", "password":"testpass"}`))

	accessToken, hasAccess := response["access_token"]
	refreshToken, hasRefresh := response["refresh_token"]

	if !hasAccess || !hasRefresh {
		t.Fatalf("Expected both tokens in response, got: %v", response)
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

	_, err := uuid.Parse(jti.(string))
	if err != nil {
		t.Errorf("%s token JTI is not a valid UUID: %v", tokenType, jti)
	}
}