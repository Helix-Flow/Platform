package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

// LegacyDatabaseManager manages database connections (renamed to avoid conflict)
type LegacyDatabaseManager struct {
	Postgres    *sql.DB
	Redis       *redis.Client
	config      *DatabaseConfig
	redisConfig *RedisConfig
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(dbConfig *DatabaseConfig, redisConfig *RedisConfig) *LegacyDatabaseManager {
	return &LegacyDatabaseManager{
		config:      dbConfig,
		redisConfig: redisConfig,
	}
}

// Initialize initializes database connections
func (dm *LegacyDatabaseManager) Initialize() error {
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
func (dm *LegacyDatabaseManager) initPostgres() error {
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
func (dm *LegacyDatabaseManager) initRedis() error {
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
func (dm *LegacyDatabaseManager) Close() error {
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
func (dm *LegacyDatabaseManager) CreateUser(username, email, password, firstName, lastName, organization string) (string, error) {
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
func (dm *LegacyDatabaseManager) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, 
			  created_at, updated_at, last_login_at, active, role 
			  FROM users WHERE username = $1`

	user := &User{}
	err := dm.Postgres.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Organization,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.Active, &user.Role,
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
func (dm *LegacyDatabaseManager) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, organization, 
			  created_at, updated_at, last_login_at, active, role 
			  FROM users WHERE email = $1`

	user := &User{}
	err := dm.Postgres.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Organization,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.Active, &user.Role,
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
func (dm *LegacyDatabaseManager) ValidatePassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// UpdateLastLogin updates user's last login time
func (dm *LegacyDatabaseManager) UpdateLastLogin(userID string) error {
	query := `UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := dm.Postgres.Exec(query, userID)
	return err
}



// GetUserPermissions retrieves user permissions
func (dm *LegacyDatabaseManager) GetUserPermissions(userID string) ([]string, error) {
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
func (dm *LegacyDatabaseManager) CreateAPIKey(userID, name, keyHash, keyPrefix string, permissions []string) (string, error) {
	query := `INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`

	var keyID string
	err := dm.Postgres.QueryRow(query, userID, name, keyHash, keyPrefix, permissions).Scan(&keyID)
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}

	return keyID, nil
}

// GetAPIKeyByHash retrieves API key by hash
func (dm *LegacyDatabaseManager) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
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
func (dm *LegacyDatabaseManager) UpdateAPIKeyUsage(keyID string) error {
	query := `UPDATE api_keys SET usage_count = usage_count + 1, last_used_at = CURRENT_TIMESTAMP 
			  WHERE id = $1`
	_, err := dm.Postgres.Exec(query, keyID)
	return err
}



// LogInferenceRequest logs an inference request
func (dm *LegacyDatabaseManager) LogInferenceRequest(userID, modelID string, requestData, responseData map[string]interface{}, status string, errorMessage *string, tokensUsed, processingTimeMs int, cost float64) error {
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

// UpdateUserProfile updates user profile information
func (dm *LegacyDatabaseManager) UpdateUserProfile(userID, firstName, lastName, organization string) error {
	query := `UPDATE users SET first_name = $1, last_name = $2, organization = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4`
	_, err := dm.Postgres.Exec(query, firstName, lastName, organization, userID)
	return err
}
