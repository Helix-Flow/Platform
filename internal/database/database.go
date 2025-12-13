package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// DatabaseManager manages database connections
type DatabaseManager struct {
	Postgres    *sql.DB
	Redis       *redis.Client
	config      *DatabaseConfig
	redisConfig *RedisConfig
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(dbConfig *DatabaseConfig, redisConfig *RedisConfig) *DatabaseManager {
	return &DatabaseManager{
		config:      dbConfig,
		redisConfig: redisConfig,
	}
}

// Initialize initializes database connections
func (dm *DatabaseManager) Initialize() error {
	// Initialize PostgreSQL connection
	if err := dm.initPostgres(); err != nil {
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Initialize Redis connection
	if err := dm.initRedis(); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	log.Println("Database connections initialized successfully")
	return nil
}

// initPostgres initializes PostgreSQL connection
func (dm *DatabaseManager) initPostgres() error {
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

	dm.Postgres = db
	return nil
}

// initRedis initializes Redis connection
func (dm *DatabaseManager) initRedis() error {
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
func (dm *DatabaseManager) Close() error {
	var errors []error

	if dm.Postgres != nil {
		if err := dm.Postgres.Close(); err != nil {
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

// CreateUser creates a new user in the database
func (dm *DatabaseManager) CreateUser(username, email, password, firstName, lastName, organization string) (string, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert user
	query := `INSERT INTO users (username, email, password_hash, first_name, last_name, organization) 
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	var userID string
	err = dm.Postgres.QueryRow(query, username, email, string(hashedPassword), firstName, lastName, organization).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// GetUserByUsername retrieves user by username
func (dm *DatabaseManager) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, 
			  created_at, updated_at, last_login, active, email_verified 
			  FROM users WHERE username = $1`

	user := &User{}
	err := dm.Postgres.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Organization,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
		&user.Active, &user.EmailVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves user by email
func (dm *DatabaseManager) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, 
			  created_at, updated_at, last_login, active, email_verified 
			  FROM users WHERE email = $1`

	user := &User{}
	err := dm.Postgres.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Organization,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
		&user.Active, &user.EmailVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// ValidatePassword validates user password
func (dm *DatabaseManager) ValidatePassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// UpdateLastLogin updates user's last login time
func (dm *DatabaseManager) UpdateLastLogin(userID string) error {
	query := `UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := dm.Postgres.Exec(query, userID)
	return err
}

// User represents a user in the database
type User struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"` // Don't include in JSON
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Organization  string     `json:"organization"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLogin     *time.Time `json:"last_login"`
	Active        bool       `json:"active"`
	EmailVerified bool       `json:"email_verified"`
}

// GetUserPermissions retrieves user permissions
func (dm *DatabaseManager) GetUserPermissions(userID string) ([]string, error) {
	query := `SELECT DISTINCT p.name 
			  FROM permissions p
			  JOIN role_permissions rp ON p.id = rp.permission_id
			  JOIN user_roles ur ON rp.role_id = ur.role_id
			  WHERE ur.user_id = $1`

	rows, err := dm.Postgres.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// CreateAPIKey creates a new API key for a user
func (dm *DatabaseManager) CreateAPIKey(userID, name, keyHash, keyPrefix string, permissions []string, expiresAt *time.Time) (string, error) {
	query := `INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions, expires_at) 
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	var keyID string
	err := dm.Postgres.QueryRow(query, userID, name, keyHash, keyPrefix, permissions, expiresAt).Scan(&keyID)
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}

	return keyID, nil
}

// GetAPIKeyByHash retrieves API key by hash
func (dm *DatabaseManager) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	query := `SELECT id, user_id, name, key_prefix, permissions, created_at, expires_at, 
			  last_used_at, usage_count, active 
			  FROM api_keys WHERE key_hash = $1 AND active = true`

	key := &APIKey{}
	err := dm.Postgres.QueryRow(query, keyHash).Scan(
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
func (dm *DatabaseManager) UpdateAPIKeyUsage(keyID string) error {
	query := `UPDATE api_keys SET usage_count = usage_count + 1, last_used_at = CURRENT_TIMESTAMP 
			  WHERE id = $1`
	_, err := dm.Postgres.Exec(query, keyID)
	return err
}

// APIKey represents an API key in the database
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	UsageCount  int        `json:"usage_count"`
	Active      bool       `json:"active"`
}

// LogInferenceRequest logs an inference request
func (dm *DatabaseManager) LogInferenceRequest(userID, modelID string, requestData, responseData map[string]interface{}, status string, errorMessage *string, tokensUsed, processingTimeMs int, cost float64) error {
	query := `INSERT INTO inference_requests 
			  (user_id, model_id, request_data, response_data, status, error_message, tokens_used, processing_time_ms, cost) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	var requestID string
	err := dm.Postgres.QueryRow(query, userID, modelID, requestData, responseData, status, errorMessage, tokensUsed, processingTimeMs, cost).Scan(&requestID)
	if err != nil {
		return fmt.Errorf("failed to log inference request: %w", err)
	}

	// Update status to completed if successful
	if status == "completed" {
		updateQuery := `UPDATE inference_requests SET completed_at = CURRENT_TIMESTAMP WHERE id = $1`
		_, err = dm.Postgres.Exec(updateQuery, requestID)
		if err != nil {
			log.Printf("Failed to update completion time for request %s: %v", requestID, err)
		}
	}

	return nil
}

// GetDefaultDatabaseConfig returns default database configuration
func GetDefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "helixflow"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "helixflow"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// GetDefaultRedisConfig returns default Redis configuration
func GetDefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnvInt("REDIS_PORT", 6379),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
	}
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
