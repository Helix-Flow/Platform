package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// SQLiteConfig holds SQLite configuration
type SQLiteConfig struct {
	DBPath string
}

// SQLiteManager manages SQLite database connections
type SQLiteManager struct {
	DB          *sql.DB
	Redis       *redis.Client
	sqliteConfig *SQLiteConfig
	redisConfig  *RedisConfig
}

// NewSQLiteManager creates a new SQLite database manager
func NewSQLiteManager(sqliteConfig *SQLiteConfig, redisConfig *RedisConfig) *SQLiteManager {
	return &SQLiteManager{
		sqliteConfig: sqliteConfig,
		redisConfig:  redisConfig,
	}
}

// Initialize initializes database connections
func (dm *SQLiteManager) Initialize() error {
	log.Printf("Initializing SQLite manager with config: %+v", dm.sqliteConfig)
	
	// Initialize SQLite connection
	if err := dm.initSQLite(); err != nil {
		return fmt.Errorf("failed to initialize SQLite: %w", err)
	}

	// Initialize Redis connection (optional)
	if err := dm.initRedis(); err != nil {
		log.Printf("Warning: Redis initialization failed: %v. Continuing without Redis.", err)
		dm.Redis = nil // Set to nil to indicate Redis is not available
	}

	log.Println("SQLite database connections initialized successfully")
	return nil
}

// initSQLite initializes SQLite connection
func (dm *SQLiteManager) initSQLite() error {
	log.Printf("Opening SQLite database at path: %s", dm.sqliteConfig.DBPath)
	
	// Ensure directory exists
	dbDir := "./data"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dm.sqliteConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	log.Printf("SQLite database opened successfully: %v", db != nil)

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	dm.DB = db
	
	// Create tables if they don't exist
	if err := dm.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	
	return nil
}

// createTables creates all necessary tables
func (dm *SQLiteManager) createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT,
			organization TEXT,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
			last_login_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			name TEXT,
			key_hash TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			permissions TEXT, -- JSON array stored as text
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			expires_at TEXT,
			last_used_at TEXT,
			usage_count INTEGER DEFAULT 0,
			active INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS inference_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id),
			model_id TEXT,
			request_size INTEGER,
			response_size INTEGER,
			latency_ms INTEGER,
			status_code INTEGER,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			ip_address TEXT,
			user_agent TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS api_usage_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id),
			api_key_id TEXT REFERENCES api_keys(id),
			method TEXT,
			path TEXT,
			status_code INTEGER,
			latency_ms INTEGER,
			request_size INTEGER,
			response_size INTEGER,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			ip_address TEXT,
			user_agent TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS system_metrics (
			id TEXT PRIMARY KEY,
			service_name TEXT,
			metric_type TEXT,
			metric_value REAL,
			unit TEXT,
			labels TEXT, -- JSON object stored as text
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			severity TEXT,
			condition TEXT,
			threshold REAL,
			current_value REAL,
			status TEXT DEFAULT 'active',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			resolved_at TEXT,
			notification_channels TEXT, -- JSON array stored as text
			labels TEXT -- JSON object stored as text
		)`,
	}

	for _, table := range tables {
		if _, err := dm.DB.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// initRedis initializes Redis connection (same as PostgreSQL version)
func (dm *SQLiteManager) initRedis() error {
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
func (dm *SQLiteManager) Close() error {
	var errors []error

	if dm.DB != nil {
		if err := dm.DB.Close(); err != nil {
			errors = append(errors, fmt.Errorf("SQLite close error: %w", err))
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

// User management functions (SQLite-compatible versions)

// CreateUser creates a new user in the database
func (dm *SQLiteManager) CreateUser(username, email, password, firstName, lastName, organization string) (string, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate UUID for user ID
	userID := generateUUID()

	// Insert user
	query := `INSERT INTO users (id, username, email, password_hash, first_name, last_name, organization) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = dm.DB.Exec(query, userID, username, email, string(hashedPassword), firstName, lastName, organization)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// GetUserByUsername retrieves user by username
func (dm *SQLiteManager) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, role, active, created_at, updated_at, last_login_at 
			  FROM users WHERE username = ?`

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
func (dm *SQLiteManager) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, role, active, created_at, updated_at, last_login_at 
			  FROM users WHERE email = ?`

	log.Printf("GetUserByEmail: searching for email '%s'", email)

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
func (dm *SQLiteManager) ValidatePassword(user *User, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

// UpdateLastLogin updates user's last login time
func (dm *SQLiteManager) UpdateLastLogin(userID string) error {
	query := `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := dm.DB.Exec(query, userID)
	return err
}

// API Key management functions

// CreateAPIKey creates a new API key
func (dm *SQLiteManager) CreateAPIKey(userID, name, keyHash, keyPrefix string, permissions []string) (string, error) {
	keyID := generateUUID()
	
	// Convert permissions to JSON
	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return "", fmt.Errorf("failed to marshal permissions: %w", err)
	}

	query := `INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, permissions, active) 
			  VALUES (?, ?, ?, ?, ?, ?, 1)`

	_, err = dm.DB.Exec(query, keyID, userID, name, keyHash, keyPrefix, string(permsJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}

	return keyID, nil
}

// GetUserPermissions retrieves user permissions
func (dm *SQLiteManager) GetUserPermissions(userID string) ([]string, error) {
	query := `SELECT permissions FROM api_keys WHERE user_id = ? AND active = 1`
	
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
func (dm *SQLiteManager) UpdateUserProfile(userID, firstName, lastName, organization string) error {
	query := `UPDATE users SET first_name = ?, last_name = ?, organization = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := dm.DB.Exec(query, firstName, lastName, organization, userID)
	return err
}

// UpdatePassword updates user password
func (dm *SQLiteManager) UpdatePassword(userID, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := dm.DB.Exec(query, passwordHash, userID)
	return err
}

// GetAPIKeyByHash retrieves API key by hash
func (dm *SQLiteManager) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	query := `SELECT id, user_id, name, key_prefix, permissions, created_at, expires_at, 
			  last_used_at, usage_count, active 
			  FROM api_keys WHERE key_hash = ? AND active = 1`

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
func (dm *SQLiteManager) UpdateAPIKeyUsage(keyID string) error {
	query := `UPDATE api_keys SET usage_count = usage_count + 1, last_used_at = CURRENT_TIMESTAMP 
			  WHERE id = ?`
	_, err := dm.DB.Exec(query, keyID)
	return err
}

// ListAPIKeys lists API keys for a user
func (dm *SQLiteManager) ListAPIKeys(userID string) ([]*APIKey, error) {
	query := `SELECT id, user_id, name, key_prefix, permissions, created_at, expires_at, 
			  last_used_at, usage_count, active 
			  FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`

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
func (dm *SQLiteManager) RevokeAPIKey(keyID string) error {
	query := `UPDATE api_keys SET active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := dm.DB.Exec(query, keyID)
	return err
}

// LogInferenceRequest logs an inference request
func (dm *SQLiteManager) LogInferenceRequest(userID, modelID string, requestData, responseData map[string]interface{}, status string, errorMessage *string, tokensUsed, processingTimeMs int, cost float64) error {
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
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = dm.DB.Exec(query, userID, modelID, string(reqJSON), string(respJSON), status, errorMessage, tokensUsed, processingTimeMs, cost)
	if err != nil {
		return fmt.Errorf("failed to log inference request: %w", err)
	}

	return nil
}