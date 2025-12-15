package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"strings"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/google/uuid"

	"helixflow/auth"
	"helixflow/database"
)

// AuthServiceServer implements the gRPC AuthService
type AuthServiceServer struct {
	auth.UnimplementedAuthServiceServer
	dbManager  database.DatabaseManager
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	blacklist  map[string]time.Time
	blMutex    sync.RWMutex
}

// NewAuthServiceServer creates a new auth service server
func NewAuthServiceServer(dbManager database.DatabaseManager) (*AuthServiceServer, error) {
	// Generate RSA keys for JWT signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &AuthServiceServer{
		dbManager:  dbManager,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		blacklist:  make(map[string]time.Time),
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

	// Create user using the database interface method
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

	log.Printf("Login attempt for username/email: %s", req.Username)
	if contains(req.Username, "@") {
		user, err = s.dbManager.GetUserByEmail(req.Username)
	} else {
		user, err = s.dbManager.GetUserByUsername(req.Username)
	}

	if err != nil {
		log.Printf("Login failed - user not found: %v", err)
		return nil, status.Error(codes.NotFound, "invalid credentials")
	}

	// Validate password
	if !s.dbManager.ValidatePassword(user, req.Password) {
		log.Printf("Login failed - invalid password for user: %s", req.Username)
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

	refreshToken, err := s.generateRefreshToken(user.ID, user.Username)
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

	// Check blacklist
	s.blMutex.RLock()
	expiry, blacklisted := s.blacklist[req.Token]
	s.blMutex.RUnlock()
	log.Printf("ValidateToken: blacklist check, token prefix: %s, blacklisted: %v, map size: %d", req.Token[:10], blacklisted, len(s.blacklist))
	if blacklisted {
		// Clean up if expired
		if time.Now().After(expiry) {
			s.blMutex.Lock()
			delete(s.blacklist, req.Token)
			s.blMutex.Unlock()
			log.Printf("Token blacklist entry expired and removed")
		} else {
			log.Printf("Token validation rejected: token revoked")
			return &auth.ValidateTokenResponse{
				Valid:   false,
				Message: "token revoked",
			}, nil
		}
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
		// Validate JTI is UUID v4
		if jti, exists := claims["jti"].(string); exists {
			parsedUUID, err := uuid.Parse(jti)
			if err != nil || uuid.Version(parsedUUID) != 4 {
				return nil, status.Error(codes.InvalidArgument, "invalid JTI format")
			}
	
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
	// Validate JTI is UUID v4
	if jti, exists := claims["jti"].(string); exists {
		parsedUUID, err := uuid.Parse(jti)
		if err != nil || parsedUUID.Version() != 4 {
			return nil, status.Error(codes.Unauthenticated, "invalid JTI format")
		}
	}
	userID := claims["sub"].(string)
	
	// Get user by ID to retrieve username
	user, err := s.dbManager.GetUserByID(userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(userID, user.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	// Generate new refresh token
	newRefreshToken, err := s.generateRefreshToken(userID, user.Username)
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
	if req.Token == "" {
		return &auth.LogoutResponse{
			Success: false,
			Message: "token is required",
		}, nil
	}

	// Parse token to get expiration
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		// If token is invalid, we still consider logout successful
		// (e.g., token already expired)
		return &auth.LogoutResponse{
			Success: true,
			Message: "logout successful",
		}, nil
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		exp, ok := claims["exp"].(float64)
		if ok {
			expiry := time.Unix(int64(exp), 0)
			s.blMutex.Lock()
			s.blacklist[req.Token] = expiry
			s.blMutex.Unlock()
			log.Printf("Token blacklisted (expires at %v)", expiry)
		}
	}

	return &auth.LogoutResponse{
		Success: true,
		Message: "logout successful",
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
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	// Update user in database
	err := s.dbManager.UpdateUserProfile(req.UserId, req.FirstName, req.LastName, req.Organization)
	if err != nil {
		log.Printf("Failed to update user profile: %v", err)
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	return &auth.UpdateUserProfileResponse{
		Success: true,
		Message: "Profile updated successfully",
	}, nil
}

// ChangePassword changes user password
func (s *AuthServiceServer) ChangePassword(ctx context.Context, req *auth.ChangePasswordRequest) (*auth.ChangePasswordResponse, error) {
	if req.UserId == "" || req.CurrentPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID, current password, and new password are required")
	}

	// Get user
	user, err := s.dbManager.GetUserByUsername(req.UserId) // Simplified - use ID in real implementation
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Validate current password
	if !s.dbManager.ValidatePassword(user, req.CurrentPassword) {
		return nil, status.Error(codes.InvalidArgument, "current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash new password")
	}

	// Update password
	err = s.dbManager.UpdatePassword(user.ID, string(hashedPassword))
	if err != nil {
		log.Printf("Failed to update password: %v", err)
		return nil, status.Error(codes.Internal, "failed to update password")
	}

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
	keyID, err := s.dbManager.CreateAPIKey(req.UserId, req.Name, keyHash, keyPrefix, req.Permissions)
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
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	dbKeys, err := s.dbManager.ListAPIKeys(req.UserId)
	if err != nil {
		log.Printf("Failed to list API keys: %v", err)
		return nil, status.Error(codes.Internal, "failed to list API keys")
	}

	var apiKeys []*auth.APIKey
	for _, dbKey := range dbKeys {
		key := &auth.APIKey{
			Id:          dbKey.ID,
			Name:        dbKey.Name,
			KeyPrefix:   dbKey.KeyPrefix,
			Permissions: dbKey.Permissions,
			CreatedAt:   dbKey.CreatedAt,
			ExpiresAt:   dbKey.ExpiresAt,
			LastUsedAt:  dbKey.LastUsedAt,
			UsageCount:  dbKey.UsageCount,
			Active:      dbKey.Active,
		}
		apiKeys = append(apiKeys, key)
	}

	return &auth.ListAPIKeysResponse{
		Success:    true,
		ApiKeys:    apiKeys,
		TotalCount: int32(len(apiKeys)),
	}, nil
}

// RevokeAPIKey revokes an API key
func (s *AuthServiceServer) RevokeAPIKey(ctx context.Context, req *auth.RevokeAPIKeyRequest) (*auth.RevokeAPIKeyResponse, error) {
	if req.KeyId == "" {
		return nil, status.Error(codes.InvalidArgument, "key ID is required")
	}

	err := s.dbManager.RevokeAPIKey(req.KeyId)
	if err != nil {
		log.Printf("Failed to revoke API key: %v", err)
		return nil, status.Error(codes.Internal, "failed to revoke API key")
	}

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
		"jti":      generateJTI(),
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"type":     "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *AuthServiceServer) generateRefreshToken(userID, username string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"jti":      generateJTI(),
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days
		"iat":      time.Now().Unix(),
		"type":     "refresh",
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
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
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
	return strings.Contains(s, substr)
}

func generateJTI() string {
	return uuid.New().String()
}
