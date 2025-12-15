package main

import (
	"testing"
	"context"
	"crypto/rsa"
	"crypto/rand"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"helixflow/auth"
	"helixflow/database"
)

var testPrivateKey *rsa.PrivateKey
var testPublicKey *rsa.PublicKey
var testServer *AuthServiceServer

// MockDatabaseManager implements DatabaseManager for testing
type MockDatabaseManager struct{}

func (m *MockDatabaseManager) Initialize() error { return nil }
func (m *MockDatabaseManager) Close() error { return nil }
func (m *MockDatabaseManager) CreateUser(username, email, password, firstName, lastName, organization string) (string, error) { return "user123", nil }
func (m *MockDatabaseManager) GetUserByUsername(username string) (*database.User, error) { 
	return &database.User{ID: "user123", Username: username, Active: true}, nil 
}
func (m *MockDatabaseManager) GetUserByEmail(email string) (*database.User, error) { 
	return &database.User{ID: "user123", Email: email, Active: true}, nil 
}
func (m *MockDatabaseManager) GetUserByID(userID string) (*database.User, error) { 
	return &database.User{ID: userID, Username: "testuser", Active: true}, nil 
}
func (m *MockDatabaseManager) ValidatePassword(user *database.User, password string) bool { return true }
func (m *MockDatabaseManager) UpdateLastLogin(userID string) error { return nil }
func (m *MockDatabaseManager) UpdateUserProfile(userID, firstName, lastName, organization string) error { return nil }
func (m *MockDatabaseManager) UpdatePassword(userID, passwordHash string) error { return nil }
func (m *MockDatabaseManager) CreateAPIKey(userID, name, keyHash, keyPrefix string, permissions []string) (string, error) { return "key123", nil }
func (m *MockDatabaseManager) GetUserPermissions(userID string) ([]string, error) { return []string{"read", "write"}, nil }
func (m *MockDatabaseManager) GetAPIKeyByHash(keyHash string) (*database.APIKey, error) { return nil, nil }
func (m *MockDatabaseManager) UpdateAPIKeyUsage(keyID string) error { return nil }
func (m *MockDatabaseManager) ListAPIKeys(userID string) ([]*database.APIKey, error) { return nil, nil }
func (m *MockDatabaseManager) RevokeAPIKey(keyID string) error { return nil }
func (m *MockDatabaseManager) LogInferenceRequest(userID, modelID string, requestData, responseData map[string]interface{}, status string, errorMessage *string, tokensUsed, processingTimeMs int, cost float64) error { return nil }

func init() {
	// Generate test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("Failed to generate test RSA key pair")
	}
	testPrivateKey = privateKey
	testPublicKey = &privateKey.PublicKey
	
	// Create test database manager
	dbManager := &MockDatabaseManager{}
	
	// Create test server
	testServer = &AuthServiceServer{
		dbManager:  dbManager,
		privateKey: testPrivateKey,
		publicKey:  testPublicKey,
		blacklist:  make(map[string]time.Time),
	}
}

func generateTestToken(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(testPrivateKey)
}

func TestValidateTokenWithValidUUIDv4(t *testing.T) {
	validUUID := uuid.New().String()
	claims := jwt.MapClaims{"jti": validUUID, "sub": "user123", "username": "testuser", "exp": float64(time.Now().Add(time.Hour).Unix())}
	tokenString, _ := generateTestToken(claims)

	req := &auth.ValidateTokenRequest{Token: tokenString}
	resp, err := testServer.ValidateToken(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !resp.Valid {
		t.Fatalf("Expected valid token, got invalid")
	}
}

func TestValidateTokenWithInvalidUUIDFormat(t *testing.T) {
	invalidUUID := "not-a-uuid"
	claims := jwt.MapClaims{"jti": invalidUUID, "sub": "user123", "username": "testuser", "exp": float64(time.Now().Add(time.Hour).Unix())}
	tokenString, _ := generateTestToken(claims)

	req := &auth.ValidateTokenRequest{Token: tokenString}
	_, err := testServer.ValidateToken(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	expectedErr := status.Error(codes.InvalidArgument, "invalid JTI format")
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestValidateTokenWithMissingJTI(t *testing.T) {
	claims := jwt.MapClaims{"sub": "user123", "username": "testuser", "exp": float64(time.Now().Add(time.Hour).Unix())}
	tokenString, _ := generateTestToken(claims)

	req := &auth.ValidateTokenRequest{Token: tokenString}
	resp, err := testServer.ValidateToken(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !resp.Valid {
		t.Fatalf("Expected valid token, got invalid. Message: %s", resp.Message)
	}
}

func TestValidateTokenWithNonV4UUID(t *testing.T) {
	nonV4UUID := "123e4567-e89b-11d3-a75a-426614174000" // UUID v1
	claims := jwt.MapClaims{"jti": nonV4UUID, "sub": "user123", "username": "testuser", "exp": float64(time.Now().Add(time.Hour).Unix())}
	tokenString, _ := generateTestToken(claims)

	req := &auth.ValidateTokenRequest{Token: tokenString}
	_, err := testServer.ValidateToken(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	expectedErr := status.Error(codes.InvalidArgument, "invalid JTI format")
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRefreshTokenWithValidUUIDv4(t *testing.T) {
	validUUID := uuid.New().String()
	claims := jwt.MapClaims{"jti": validUUID, "sub": "user123", "username": "testuser", "exp": float64(time.Now().Add(time.Hour).Unix())}
	tokenString, _ := generateTestToken(claims)

	req := &auth.RefreshTokenRequest{RefreshToken: tokenString}
	_, err := testServer.RefreshToken(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRefreshTokenWithInvalidUUIDFormat(t *testing.T) {
	invalidUUID := "not-a-uuid"
	claims := jwt.MapClaims{"jti": invalidUUID, "sub": "user123", "username": "testuser", "exp": float64(time.Now().Add(time.Hour).Unix())}
	tokenString, _ := generateTestToken(claims)

	req := &auth.RefreshTokenRequest{RefreshToken: tokenString}
	_, err := testServer.RefreshToken(context.Background(), req)
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
	claims := jwt.MapClaims{"jti": nonV4UUID, "sub": "user123", "username": "testuser", "exp": float64(time.Now().Add(time.Hour).Unix())}
	tokenString, _ := generateTestToken(claims)

	req := &auth.RefreshTokenRequest{RefreshToken: tokenString}
	_, err := testServer.RefreshToken(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	expectedErr := status.Error(codes.Unauthenticated, "invalid JTI format")
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
}