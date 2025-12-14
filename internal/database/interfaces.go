package database

import (
	"context"
	"fmt"
	"time"
)

// User represents a user in the system
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Organization string
	Role         string
	Active       bool
	CreatedAt    string
	UpdatedAt    string
	LastLoginAt  string
}

// APIKey represents an API key
type APIKey struct {
	ID                  string
	UserID              string
	Name                string
	KeyHash             string
	KeyPrefix           string
	Permissions         []string
	CreatedAt           string
	ExpiresAt           string
	LastUsedAt          string
	UsageCount          int32
	Active              bool
}

// DatabaseManager interface defines the contract for database operations
type DatabaseManager interface {
	Initialize() error
	Close() error
	
	// User management
	CreateUser(username, email, password, firstName, lastName, organization string) (string, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(userID string) (*User, error)
	ValidatePassword(user *User, password string) bool
	UpdateLastLogin(userID string) error
	UpdateUserProfile(userID, firstName, lastName, organization string) error
	UpdatePassword(userID, passwordHash string) error
	
	// API Key management
	CreateAPIKey(userID, name, keyHash, keyPrefix string, permissions []string) (string, error)
	GetUserPermissions(userID string) ([]string, error)
	GetAPIKeyByHash(keyHash string) (*APIKey, error)
	UpdateAPIKeyUsage(keyID string) error
	ListAPIKeys(userID string) ([]*APIKey, error)
	RevokeAPIKey(keyID string) error
	
	// Inference logging
	LogInferenceRequest(userID, modelID string, requestData, responseData map[string]interface{}, status string, errorMessage *string, tokensUsed, processingTimeMs int, cost float64) error
}

// DataStore interface for database operations (unified interface)
type DataStore interface {
	Initialize() error
	Close() error
	
	// User management
	CreateUser(username, email, password, firstName, lastName, organization string) (string, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	ValidatePassword(user *User, password string) bool
	UpdateLastLogin(userID string) error
	
	// API Key management
	CreateAPIKey(userID, name, keyHash, keyPrefix string, permissions []string) (string, error)
	GetUserPermissions(userID string) ([]string, error)
}

// generateUUID generates a simple UUID for testing
func generateUUID() string {
	return fmt.Sprintf("%d-%d-%d-%d-%d", 
		time.Now().UnixNano(), 
		time.Now().UnixNano()>>32, 
		time.Now().UnixNano()>>16, 
		time.Now().UnixNano()>>8, 
		time.Now().UnixNano())
}

// AuthService interface for authentication operations
type AuthService interface {
	Register(ctx context.Context, username, email, password, firstName, lastName, organization string) (string, error)
	Login(ctx context.Context, usernameOrEmail, password string) (*User, string, error)
	ValidateToken(ctx context.Context, token string) (*User, error)
	CreateAPIKey(ctx context.Context, userID, name string, permissions []string) (*APIKey, error)
}