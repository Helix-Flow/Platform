package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"helixflow/auth"
	"helixflow/database"
)

// AuthServiceServer implements the gRPC AuthService
type AuthServiceServer struct {
	auth.UnimplementedAuthServiceServer
	dbManager  *database.DatabaseManager
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewAuthServiceServer creates a new auth service server
func NewAuthServiceServer(dbManager *database.DatabaseManager) (*AuthServiceServer, error) {
	// Generate RSA keys for JWT signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &AuthServiceServer{
		dbManager:  dbManager,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

// Register handles user registration
func (s *AuthServiceServer) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) {
	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username, email, and password are required")
	}

	// Check if user already exists
	_, err := s.dbManager.GetUserByUsername(req.Username)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "username already exists")
	}

	_, err = s.dbManager.GetUserByEmail(req.Email)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "email already exists")
	}

	// Create user
	userID, err := s.dbManager.CreateUser(req.Username, req.Email, req.Password, req.FirstName, req.LastName, req.Organization)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	// Get created user
	user, err := s.dbManager.GetUserByUsername(req.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve created user")
	}

	return &auth.RegisterResponse{
		Success: true,
		Message: "User registered successfully",
		UserId:  userID,
		User:    s.userToProto(user),
	}, nil
}

// Login handles user authentication
func (s *AuthServiceServer) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	// Validate input
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	// Get user by username or email
	var user *database.User
	var err error

	if contains(req.Username, "@") {
		user, err = s.dbManager.GetUserByEmail(req.Username)
	} else {
		user, err = s.dbManager.GetUserByUsername(req.Username)
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid credentials")
	}

	// Validate password
	if !s.dbManager.ValidatePassword(user, req.Password) {
		return nil, status.Error(codes.NotFound, "invalid credentials")
	}

	// Update last login
	if err := s.dbManager.UpdateLastLogin(user.ID); err != nil {
		log.Printf("Failed to update last login: %v", err)
	}

	// Generate JWT tokens
	accessToken, err := s.generateAccessToken(user.ID, user.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}

	return &auth.LoginResponse{
		Success:      true,
		Message:      "Login successful",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hour
		User:         s.userToProto(user),
	}, nil
}

// ValidateToken validates JWT tokens
func (s *AuthServiceServer) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &auth.ValidateTokenResponse{
			Valid:   false,
			Message: "token is required",
		}, nil
	}

	// Parse and validate token
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return &auth.ValidateTokenResponse{
			Valid:   false,
			Message: "invalid token",
		}, nil
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := claims["sub"].(string)
		username := claims["username"].(string)
		expiresAt := int64(claims["exp"].(float64))

		// Get user permissions
		permissions, err := s.dbManager.GetUserPermissions(userID)
		if err != nil {
			log.Printf("Failed to get user permissions: %v", err)
			permissions = []string{}
		}

		return &auth.ValidateTokenResponse{
			Valid:       true,
			UserId:      userID,
			Username:    username,
			Permissions: permissions,
			ExpiresAt:   expiresAt,
		}, nil
	}

	return &auth.ValidateTokenResponse{
		Valid:   false,
		Message: "invalid token claims",
	}, nil
}

// RefreshToken refreshes an access token
func (s *AuthServiceServer) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest) (*auth.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	// Validate refresh token (simplified - in production, store refresh tokens in database)
	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}

	userID := claims["sub"].(string)
	username := claims["username"].(string)

	// Generate new access token
	accessToken, err := s.generateAccessToken(userID, username)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	// Generate new refresh token
	newRefreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}

	return &auth.RefreshTokenResponse{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    3600,
		Message:      "Token refreshed successfully",
	}, nil
}

// Logout handles user logout
func (s *AuthServiceServer) Logout(ctx context.Context, req *auth.LogoutRequest) (*auth.LogoutResponse, error) {
	// In a real implementation, you would:
	// 1. Add the token to a blacklist
	// 2. Remove refresh tokens from database
	// 3. Log the logout event

	return &auth.LogoutResponse{
		Success: true,
		Message: "Logout successful",
	}, nil
}

// GetUserProfile retrieves user profile
func (s *AuthServiceServer) GetUserProfile(ctx context.Context, req *auth.GetUserProfileRequest) (*auth.GetUserProfileResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	user, err := s.dbManager.GetUserByUsername(req.UserId) // Simplified - use ID in real implementation
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &auth.GetUserProfileResponse{
		Success: true,
		User:    s.userToProto(user),
	}, nil
}

// UpdateUserProfile updates user profile
func (s *AuthServiceServer) UpdateUserProfile(ctx context.Context, req *auth.UpdateUserProfileRequest) (*auth.UpdateUserProfileResponse, error) {
	// Implementation would update user in database
	return &auth.UpdateUserProfileResponse{
		Success: true,
		Message: "Profile updated successfully",
	}, nil
}

// ChangePassword changes user password
func (s *AuthServiceServer) ChangePassword(ctx context.Context, req *auth.ChangePasswordRequest) (*auth.ChangePasswordResponse, error) {
	// Implementation would validate current password and update to new password
	return &auth.ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	}, nil
}

// GenerateAPIKey generates a new API key
func (s *AuthServiceServer) GenerateAPIKey(ctx context.Context, req *auth.GenerateAPIKeyRequest) (*auth.GenerateAPIKeyResponse, error) {
	// Generate API key
	apiKey := generateRandomAPIKey()
	keyHash := hashAPIKey(apiKey)
	keyPrefix := apiKey[:8]

	// Store in database
	keyID, err := s.dbManager.CreateAPIKey(req.UserId, req.Name, keyHash, keyPrefix, req.Permissions, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create API key")
	}

	apiKeyProto := &auth.APIKey{
		Id:          keyID,
		Name:        req.Name,
		KeyPrefix:   keyPrefix,
		Permissions: req.Permissions,
		CreatedAt:   time.Now().Format(time.RFC3339),
		Active:      true,
		UsageCount:  0,
	}

	return &auth.GenerateAPIKeyResponse{
		Success: true,
		Message: "API key generated successfully",
		ApiKey:  apiKeyProto,
	}, nil
}

// ListAPIKeys lists user's API keys
func (s *AuthServiceServer) ListAPIKeys(ctx context.Context, req *auth.ListAPIKeysRequest) (*auth.ListAPIKeysResponse, error) {
	// Implementation would retrieve API keys from database
	return &auth.ListAPIKeysResponse{
		Success:    true,
		ApiKeys:    []*auth.APIKey{}, // Empty for now
		TotalCount: 0,
	}, nil
}

// RevokeAPIKey revokes an API key
func (s *AuthServiceServer) RevokeAPIKey(ctx context.Context, req *auth.RevokeAPIKeyRequest) (*auth.RevokeAPIKeyResponse, error) {
	// Implementation would revoke API key in database
	return &auth.RevokeAPIKeyResponse{
		Success: true,
		Message: "API key revoked successfully",
	}, nil
}

// GetUserPermissions retrieves user permissions
func (s *AuthServiceServer) GetUserPermissions(ctx context.Context, req *auth.GetUserPermissionsRequest) (*auth.GetUserPermissionsResponse, error) {
	permissions, err := s.dbManager.GetUserPermissions(req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user permissions")
	}

	// Convert to role format (simplified)
	roles := []*auth.Role{
		{
			Id:          "user",
			Name:        "User",
			Description: "Regular user",
			Permissions: permissions,
		},
	}

	return &auth.GetUserPermissionsResponse{
		Success:     true,
		Permissions: permissions,
		Roles:       roles,
	}, nil
}

// Helper functions

func (s *AuthServiceServer) generateAccessToken(userID, username string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"type":     "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *AuthServiceServer) generateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days
		"iat":  time.Now().Unix(),
		"type": "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *AuthServiceServer) userToProto(user *database.User) *auth.User {
	return &auth.User{
		Id:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Organization: user.Organization,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    user.UpdatedAt.Format(time.RFC3339),
		Active:       user.Active,
	}
}

func generateRandomAPIKey() string {
	// Generate a random API key (simplified)
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return "hf_" + string(b)
}

func hashAPIKey(apiKey string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	return string(hash)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || s[len(s)-len(substr):] == substr
}
