package main

import (
	"testing"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testKey = []byte("test-secret")

func generateTestToken(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(testKey)
}

func TestValidateTokenWithValidUUIDv4(t *testing.T) {
	validUUID := uuid.New().String()
	claims := jwt.MapClaims{"jti": validUUID, "sub": "user123"}
	tokenString, _ := generateTestToken(claims)

	_, err := ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateTokenWithInvalidUUIDFormat(t *testing.T) {
	invalidUUID := "not-a-uuid"
	claims := jwt.MapClaims{"jti": invalidUUID, "sub": "user123"}
	tokenString, _ := generateTestToken(claims)

	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	expectedErr := status.Error(codes.Unauthenticated, "invalid JTI format")
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestValidateTokenWithMissingJTI(t *testing.T) {
	claims := jwt.MapClaims{"sub": "user123"}
	tokenString, _ := generateTestToken(claims)

	_, err := ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestValidateTokenWithNonV4UUID(t *testing.T) {
	nonV4UUID := "123e4567-e89b-11d3-a75a-426614174000" // UUID v1
	claims := jwt.MapClaims{"jti": nonV4UUID, "sub": "user123"}
	tokenString, _ := generateTestToken(claims)

	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	expectedErr := status.Error(codes.Unauthenticated, "invalid JTI format")
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRefreshTokenWithValidUUIDv4(t *testing.T) {
	validUUID := uuid.New().String()
	claims := jwt.MapClaims{"jti": validUUID, "sub": "user123"}
	tokenString, _ := generateTestToken(claims)

	_, err := RefreshToken(tokenString)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRefreshTokenWithInvalidUUIDFormat(t *testing.T) {
	invalidUUID := "not-a-uuid"
	claims := jwt.MapClaims{"jti": invalidUUID, "sub": "user123"}
	tokenString, _ := generateTestToken(claims)

	_, err := RefreshToken(tokenString)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	expectedErr := status.Error(codes.Unauthenticated, "invalid JTI format")
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRefreshTokenWithNonV4UUID(t *testing.T) {
	nonV4UUID := "123e4567-e89b-11d3-a75a-426614174000" // UUID v1
	claims := jwt.MapClaims{"jti": nonV4UUID, "sub": "user123"}
	tokenString, _ := generateTestToken(claims)

	_, err := RefreshToken(tokenString)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	expectedErr := status.Error(codes.Unauthenticated, "invalid JTI format")
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}