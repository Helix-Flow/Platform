package database

import (
	"os"
	"strconv"
)

// DatabaseType represents the type of database to use
type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeSQLite   DatabaseType = "sqlite"
)

// GetDatabaseType returns the database type from environment
func GetDatabaseType() DatabaseType {
	dbType := os.Getenv("DATABASE_TYPE")
	if dbType == "" {
		// Default to SQLite for development
		return DatabaseTypeSQLite
	}
	return DatabaseType(dbType)
}

// GetDatabaseConfig returns database configuration based on type
func GetDatabaseConfig() interface{} {
	dbType := GetDatabaseType()
	
	switch dbType {
	case DatabaseTypePostgres:
		return GetPostgreSQLConfig()
	case DatabaseTypeSQLite:
		return GetSQLiteConfig()
	default:
		return GetSQLiteConfig() // Default to SQLite
	}
}

// GetPostgreSQLConfig returns PostgreSQL configuration
func GetPostgreSQLConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "helixflow"),
		Password: getEnv("DB_PASSWORD", "helixflow123"),
		DBName:   getEnv("DB_NAME", "helixflow"),
		SSLMode:  getEnv("DB_SSL_MODE", "require"),
	}
}

// GetSQLiteConfig returns SQLite configuration
func GetSQLiteConfig() *SQLiteConfig {
	return &SQLiteConfig{
		DBPath: getEnv("DB_PATH", "./data/helixflow.db"),
	}
}

// GetRedisConfig returns Redis configuration
func GetRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnvInt("REDIS_PORT", 6379),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
	}
}

// GetDefaultDatabaseConfig returns default database configuration (SQLite for development)
func GetDefaultDatabaseConfig() interface{} {
	return GetSQLiteConfig()
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

// Helper functions
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