package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// PostgresManager manages PostgreSQL database connections
type PostgresManager struct {
	DB          *sql.DB
	Redis       *redis.Client
	config      *DatabaseConfig
	redisConfig *RedisConfig
}

// NewPostgresManager creates a new PostgreSQL database manager
func NewPostgresManager(config *DatabaseConfig, redisConfig *RedisConfig) *PostgresManager {
	return &PostgresManager{
		config:      config,
		redisConfig: redisConfig,
	}
}

// Initialize initializes database connections
func (dm *PostgresManager) Initialize() error {
	// Initialize PostgreSQL connection
	if err := dm.initPostgres(); err != nil {
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Initialize Redis connection (optional)
	if err := dm.initRedis(); err != nil {
		log.Printf("Warning: Redis initialization failed: %v. Continuing without Redis.", err)
		dm.Redis = nil // Set to nil to indicate Redis is not available
	}

	log.Println("PostgreSQL database connections initialized successfully")
	return nil
}

// initPostgres initializes PostgreSQL connection
func (dm *PostgresManager) initPostgres() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dm.config.Host, dm.config.Port, dm.config.User, dm.config.Password, dm.config.DBName, dm.config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	dm.DB = db
	return nil
}

// initRedis initializes Redis connection
func (dm *PostgresManager) initRedis() error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", dm.redisConfig.Host, dm.redisConfig.Port),
		Password: dm.redisConfig.Password,
		DB:       dm.redisConfig.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return err
	}

	dm.Redis = rdb
	return nil
}

// Close closes database connections
func (dm *PostgresManager) Close() error {
	var errors []error

	if dm.DB != nil {
		if err := dm.DB.Close(); err != nil {
			errors = append(errors, fmt.Errorf("PostgreSQL close error: %w", err))
		}
	}

	if dm.Redis != nil {
		if err := dm.Redis.Close(); err != nil {
			errors = append(errors, fmt.Errorf("Redis close error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing databases: %v", errors)
	}

	return nil
}

// User management functions (interface-compatible versions)

// CreateUser creates a new user in the database
func (dm *PostgresManager) CreateUser(username, email, password, firstName, lastName, organization string) (string, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate UUID for user ID
	userID := generateUUID()

	// Insert user
	query := `INSERT INTO users (id, username, email, password_hash, first_name, last_name, organization) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = dm.DB.Exec(query, userID, username, email, string(hashedPassword), firstName, lastName, organization)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// GetUserByUsername retrieves user by username
func (dm *PostgresManager) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, role, active, 
			  to_char(created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at,
			  to_char(updated_at, 'YYYY-MM-DD HH24:MI:SS') as updated_at,
			  to_char(last_login_at, 'YYYY-MM-DD HH24:MI:SS') as last_login_at
			  FROM users WHERE username = $1`

	var user User
	var createdAt, updatedAt, lastLoginAt sql.NullString
	
	err := dm.DB.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Organization, &user.Role,
		&user.Active, &createdAt, &updatedAt, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt = createdAt.String
	user.UpdatedAt = updatedAt.String
	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.String
	}

	return &user, nil
}

// GetUserByEmail retrieves user by email
func (dm *PostgresManager) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, role, active,
			  to_char(created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at,
			  to_char(updated_at, 'YYYY-MM-DD HH24:MI:SS') as updated_at,
			  to_char(last_login_at, 'YYYY-MM-DD HH24:MI:SS') as last_login_at
			  FROM users WHERE email = $1`

	var user User
	var createdAt, updatedAt, lastLoginAt sql.NullString
	
	err := dm.DB.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Organization, &user.Role,
		&user.Active, &createdAt, &updatedAt, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt = createdAt.String
	user.UpdatedAt = updatedAt.String
	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.String
	}

	return &user, nil
}

// ValidatePassword validates user password
func (dm *PostgresManager) ValidatePassword(user *User, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

// UpdateLastLogin updates user's last login time
func (dm *PostgresManager) UpdateLastLogin(userID string) error {
	query := `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := dm.DB.Exec(query, userID)
	return err
}

// CreateAPIKey creates a new API key
func (dm *PostgresManager) CreateAPIKey(userID, name, keyHash, keyPrefix string, permissions []string) (string, error) {
	keyID := generateUUID()
	
	// Convert permissions to JSON
	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return "", fmt.Errorf("failed to marshal permissions: %w", err)
	}

	query := `INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, permissions, active) 
			  VALUES ($1, $2, $3, $4, $5, $6, true)`

	_, err = dm.DB.Exec(query, keyID, userID, name, keyHash, keyPrefix, string(permsJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}

	return keyID, nil
}

// GetUserPermissions retrieves user permissions
func (dm *PostgresManager) GetUserPermissions(userID string) ([]string, error) {
	query := `SELECT DISTINCT permissions FROM api_keys WHERE user_id = $1 AND active = true`
	
	rows, err := dm.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}
	defer rows.Close()

	var allPermissions []string
	for rows.Next() {
		var permsJSON string
		if err := rows.Scan(&permsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan permissions: %w", err)
		}

		var permissions []string
		if err := json.Unmarshal([]byte(permsJSON), &permissions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
		}

		allPermissions = append(allPermissions, permissions...)
	}

	return allPermissions, nil
}

// UpdateUserProfile updates user profile information
func (dm *PostgresManager) UpdateUserProfile(userID, firstName, lastName, organization string) error {
	query := `UPDATE users SET first_name = $1, last_name = $2, organization = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4`
	_, err := dm.DB.Exec(query, firstName, lastName, organization, userID)
	return err
}

// UpdatePassword updates user password
func (dm *PostgresManager) UpdatePassword(userID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := dm.DB.Exec(query, passwordHash, userID)
	return err
}

// GetAPIKeyByHash retrieves API key by hash
func (dm *PostgresManager) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	query := `SELECT id, user_id, name, key_prefix, permissions, created_at, expires_at, 
			  last_used_at, usage_count, active 
			  FROM api_keys WHERE key_hash = $1 AND active = true`

	key := &APIKey{}
	err := dm.DB.QueryRow(query, keyHash).Scan(
		&key.ID, &key.UserID, &key.Name, &key.KeyPrefix, &key.Permissions,
		&key.CreatedAt, &key.ExpiresAt, &key.LastUsedAt, &key.UsageCount, &key.Active,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return key, nil
}

// UpdateAPIKeyUsage updates API key usage information
func (dm *PostgresManager) UpdateAPIKeyUsage(keyID string) error {
	query := `UPDATE api_keys SET usage_count = usage_count + 1, last_used_at = CURRENT_TIMESTAMP 
			  WHERE id = $1`
	_, err := dm.DB.Exec(query, keyID)
	return err
}

// ListAPIKeys lists API keys for a user
func (dm *PostgresManager) ListAPIKeys(userID string) ([]*APIKey, error) {
	query := `SELECT id, user_id, name, key_prefix, permissions, created_at, expires_at, 
			  last_used_at, usage_count, active 
			  FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := dm.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*APIKey
	for rows.Next() {
		key := &APIKey{}
		err := rows.Scan(
			&key.ID, &key.UserID, &key.Name, &key.KeyPrefix, &key.Permissions,
			&key.CreatedAt, &key.ExpiresAt, &key.LastUsedAt, &key.UsageCount, &key.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, key)
	}

	return apiKeys, nil
}

// RevokeAPIKey revokes an API key
func (dm *PostgresManager) RevokeAPIKey(keyID string) error {
	query := `UPDATE api_keys SET active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := dm.DB.Exec(query, keyID)
	return err
}

// LogInferenceRequest logs an inference request
func (dm *PostgresManager) LogInferenceRequest(userID, modelID string, requestData, responseData map[string]interface{}, status string, errorMessage *string, tokensUsed, processingTimeMs int, cost float64) error {
	// Convert maps to JSON
	reqJSON, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	respJSON, err := json.Marshal(responseData)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	query := `INSERT INTO inference_requests 
			  (user_id, model_id, request_data, response_data, status, error_message, tokens_used, processing_time_ms, cost) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	var requestID string
	err = dm.DB.QueryRow(query, userID, modelID, string(reqJSON), string(respJSON), status, errorMessage, tokensUsed, processingTimeMs, cost).Scan(&requestID)
	if err != nil {
		return fmt.Errorf("failed to log inference request: %w", err)
	}

	// Update status to completed if successful
	if status == "completed" {
		updateQuery := `UPDATE inference_requests SET completed_at = CURRENT_TIMESTAMP WHERE id = $1`
		_, err = dm.DB.Exec(updateQuery, requestID)
		if err != nil {
			log.Printf("Failed to update completion time for request %s: %v", requestID, err)
		}
	}

	return nil
}