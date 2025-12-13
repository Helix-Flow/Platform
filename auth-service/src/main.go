package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	redisClient *redis.Client
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
}

type User struct {
	UserID           string `json:"user_id"`
	Email            string `json:"email"`
	SubscriptionTier string `json:"subscription_tier"`
	APIKeyHash       string `json:"api_key_hash"`
	Status           string `json:"status"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type APIKey struct {
	KeyID     string     `json:"key_id"`
	UserID    string     `json:"user_id"`
	KeyHash   string     `json:"key_hash"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RateLimit int        `json:"rate_limit"`
	Status    string     `json:"status"`
}

func NewAuthService() *AuthService {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_HOST", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Load RSA keys
	privateKey := loadPrivateKey("/secrets/jwt-private.pem")
	publicKey := loadPublicKey("/secrets/jwt-public.pem")

	return &AuthService{
		redisClient: rdb,
		privateKey:  privateKey,
		publicKey:   publicKey,
	}
}

func (as *AuthService) AuthenticateUser(email, password string) (*User, error) {
	// Get user from database (placeholder - integrate with actual DB)
	userData, err := as.getUserByEmail(email)
	if err != nil || userData == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(userData.APIKeyHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return userData, nil
}

func (as *AuthService) GenerateTokens(user *User) (*TokenPair, error) {
	now := time.Now()

	// Access token
	accessClaims := jwt.MapClaims{
		"sub":   user.UserID,
		"email": user.Email,
		"tier":  user.SubscriptionTier,
		"type":  "access",
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(as.privateKey)
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshClaims := jwt.MapClaims{
		"sub":  user.UserID,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(30 * 24 * time.Hour).Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(as.privateKey)
	if err != nil {
		return nil, err
	}

	// Store refresh token in Redis
	err = as.redisClient.Set(context.Background(),
		fmt.Sprintf("refresh_token:%s", user.UserID),
		refreshTokenString,
		30*24*time.Hour).Err()
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    15 * 60,
	}, nil
}

func (as *AuthService) ValidateToken(tokenString, tokenType string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["type"] != tokenType {
			return nil, fmt.Errorf("invalid token type")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (as *AuthService) RefreshAccessToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := as.ValidateToken(refreshTokenString, "refresh")
	if err != nil {
		return nil, err
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	// Verify refresh token exists in Redis
	storedToken, err := as.redisClient.Get(context.Background(),
		fmt.Sprintf("refresh_token:%s", userID)).Result()
	if err != nil || storedToken != refreshTokenString {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Get user data (placeholder)
	userData, err := as.getUserByID(userID)
	if err != nil {
		return nil, err
	}

	return as.GenerateTokens(userData)
}

func (as *AuthService) GenerateAPIKey(userID, name string, rateLimit int, expiresDays int) (string, string, error) {
	// Generate random key
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return "", "", err
	}

	apiKey := fmt.Sprintf("hf_%s", encodeBase62(keyBytes))

	// Generate key ID
	keyIDBytes := make([]byte, 16)
	_, err = rand.Read(keyIDBytes)
	if err != nil {
		return "", "", err
	}
	keyID := encodeBase62(keyIDBytes)

	// Hash the key
	keyHash := hashKey(apiKey)

	// Create key record
	now := time.Now()
	expiresAt := now.AddDate(0, 0, expiresDays)
	apiKeyRecord := APIKey{
		KeyID:     keyID,
		UserID:    userID,
		KeyHash:   keyHash,
		Name:      name,
		CreatedAt: now,
		ExpiresAt: &expiresAt,
		RateLimit: rateLimit,
		Status:    "active",
	}

	// Store in Redis
	keyJSON, err := json.Marshal(apiKeyRecord)
	if err != nil {
		return "", "", err
	}

	err = as.redisClient.Set(context.Background(),
		fmt.Sprintf("api_key:%s", keyID),
		string(keyJSON),
		time.Until(expiresAt)).Err()
	if err != nil {
		return "", "", err
	}

	// Index by user
	err = as.redisClient.SAdd(context.Background(),
		fmt.Sprintf("user_keys:%s", userID), keyID).Err()
	if err != nil {
		return "", "", err
	}

	return keyID, apiKey, nil
}

func (as *AuthService) ValidateAPIKey(apiKey string) (*APIKey, error) {
	// Hash the provided key
	keyHash := hashKey(apiKey)

	// Find key by hash (inefficient - use database in production)
	keys, err := as.redisClient.Keys(context.Background(), "api_key:*").Result()
	if err != nil {
		return nil, err
	}

	for _, keyKey := range keys {
		keyData, err := as.redisClient.Get(context.Background(), keyKey).Result()
		if err != nil {
			continue
		}

		var keyRecord APIKey
		err = json.Unmarshal([]byte(keyData), &keyRecord)
		if err != nil {
			continue
		}

		if keyRecord.KeyHash == keyHash {
			// Check expiration
			if keyRecord.ExpiresAt != nil && time.Now().After(*keyRecord.ExpiresAt) {
				keyRecord.Status = "expired"
				as.updateAPIKey(&keyRecord)
				return nil, fmt.Errorf("key expired")
			}

			// Check status
			if keyRecord.Status != "active" {
				return nil, fmt.Errorf("key inactive")
			}

			// Update last used
			now := time.Now()
			keyRecord.LastUsed = &now
			as.updateAPIKey(&keyRecord)

			return &keyRecord, nil
		}
	}

	return nil, fmt.Errorf("key not found")
}

func (as *AuthService) CheckRateLimit(keyID string, rateLimit int) bool {
	// Simple rate limiting - in production use more sophisticated approach
	currentMinute := time.Now().Truncate(time.Minute)
	windowKey := fmt.Sprintf("rate_limit:%s:%s", keyID, currentMinute.Format(time.RFC3339))

	count, err := as.redisClient.Incr(context.Background(), windowKey).Result()
	if err != nil {
		return true // Allow on error
	}

	// Set expiration
	as.redisClient.Expire(context.Background(), windowKey, time.Minute)

	return int(count) <= rateLimit
}

func (as *AuthService) updateAPIKey(key *APIKey) error {
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return err
	}

	var expiration time.Duration
	if key.ExpiresAt != nil {
		expiration = time.Until(*key.ExpiresAt)
	}

	return as.redisClient.Set(context.Background(),
		fmt.Sprintf("api_key:%s", key.KeyID),
		string(keyJSON),
		expiration).Err()
}

func (as *AuthService) getUserByEmail(email string) (*User, error) {
	// Placeholder - integrate with actual database
	if email == "test@example.com" {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		return &User{
			UserID:           "user-123",
			Email:            email,
			SubscriptionTier: "PRO",
			APIKeyHash:       string(hashedPassword),
			Status:           "ACTIVE",
		}, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (as *AuthService) getUserByID(userID string) (*User, error) {
	// Placeholder - integrate with actual database
	if userID == "user-123" {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		return &User{
			UserID:           userID,
			Email:            "test@example.com",
			SubscriptionTier: "PRO",
			APIKeyHash:       string(hashedPassword),
			Status:           "ACTIVE",
		}, nil
	}
	return nil, fmt.Errorf("user not found")
}

// HTTP Handlers
func (as *AuthService) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := as.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	tokens, err := as.GenerateTokens(user)
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

func (as *AuthService) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tokens, err := as.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		http.Error(w, "Token refresh failed", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

func (as *AuthService) generateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID      string `json:"user_id"`
		Name        string `json:"name"`
		RateLimit   int    `json:"rate_limit"`
		ExpiresDays int    `json:"expires_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.RateLimit == 0 {
		req.RateLimit = 60
	}
	if req.ExpiresDays == 0 {
		req.ExpiresDays = 365
	}

	keyID, apiKey, err := as.GenerateAPIKey(req.UserID, req.Name, req.RateLimit, req.ExpiresDays)
	if err != nil {
		http.Error(w, "API key generation failed", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"key_id":  keyID,
		"api_key": apiKey,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (as *AuthService) validateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	keyRecord, err := as.ValidateAPIKey(apiKey)
	if err != nil {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	// Check rate limit
	if !as.CheckRateLimit(keyRecord.KeyID, keyRecord.RateLimit) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keyRecord)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "auth-service",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Utility functions
func loadPrivateKey(filename string) *rsa.PrivateKey {
	keyData, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		log.Fatal("Failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	return privateKey
}

func loadPublicKey(filename string) *rsa.PublicKey {
	keyData, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to load public key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		log.Fatal("Failed to decode PEM block")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse public key: %v", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		log.Fatal("Not an RSA public key")
	}

	return rsaPublicKey
}

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

func encodeBase62(data []byte) string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	result := ""
	num := uint64(0)
	for _, b := range data {
		num = num<<8 + uint64(b)
	}

	if num == 0 {
		return "0"
	}

	for num > 0 {
		result = string(charset[num%62]) + result
		num /= 62
	}

	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	authService := NewAuthService()

	r := mux.NewRouter()

	// Routes
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/login", authService.loginHandler).Methods("POST")
	r.HandleFunc("/refresh", authService.refreshHandler).Methods("POST")
	r.HandleFunc("/api-keys/generate", authService.generateAPIKeyHandler).Methods("POST")
	r.HandleFunc("/api-keys/validate", authService.validateAPIKeyHandler).Methods("GET")

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Starting auth service on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
