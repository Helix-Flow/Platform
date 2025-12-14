package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// PostgresAdvancedManager extends PostgresManager with enterprise features
type PostgresAdvancedManager struct {
	*PostgresManager
	
	// Connection pooling
	primaryDB     *sql.DB
	readReplicaDB *sql.DB
	
	// Advanced features
	queryCache    *QueryCache
	connectionPool *ConnectionPool
	
	// Monitoring
	metricsCollector *MetricsCollector
	
	// Configuration
	advancedConfig *AdvancedDatabaseConfig
}

// AdvancedDatabaseConfig holds advanced database configuration
type AdvancedDatabaseConfig struct {
	// Connection pooling
	MaxConnections       int           `json:"max_connections"`
	MinConnections       int           `json:"min_connections"`
	ConnectionTimeout    time.Duration `json:"connection_timeout"`
	IdleTimeout          time.Duration `json:"idle_timeout"`
	MaxLifetime          time.Duration `json:"max_lifetime"`
	
	// Read replicas
	ReadReplicaEnabled   bool          `json:"read_replica_enabled"`
	ReadReplicaHost      string        `json:"read_replica_host"`
	ReadReplicaPort      int           `json:"read_replica_port"`
	
	// Caching
	QueryCacheEnabled    bool          `json:"query_cache_enabled"`
	QueryCacheSize       int           `json:"query_cache_size"`
	QueryCacheTTL        time.Duration `json:"query_cache_ttl"`
	
	// Performance
	StatementTimeout     time.Duration `json:"statement_timeout"`
	LockTimeout          time.Duration `json:"lock_timeout"`
	IdleInTransactionTimeout time.Duration `json:"idle_in_transaction_timeout"`
	
	// Monitoring
	MetricsEnabled       bool          `json:"metrics_enabled"`
	SlowQueryThreshold   time.Duration `json:"slow_query_threshold"`
}

// QueryCache provides in-memory query result caching
type QueryCache struct {
	cache    map[string]*CacheEntry
	mutex    sync.RWMutex
	maxSize  int
	ttl      time.Duration
}

// CacheEntry represents a cached query result
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
}

// ConnectionPool manages database connection pooling
type ConnectionPool struct {
	primaryPool   *sql.DB
	replicaPool   *sql.DB
	maxConnections int
	minConnections int
	stats         *PoolStats
	mutex         sync.RWMutex
}

// PoolStats tracks connection pool statistics
type PoolStats struct {
	ActiveConnections   int32
	IdleConnections     int32
	TotalConnections    int32
	WaitTime            time.Duration
	ConnectionTime      time.Duration
	Timeouts            int32
}

// MetricsCollector collects database performance metrics
type MetricsCollector struct {
	queryCount        int64
	errorCount        int64
	slowQueryCount    int64
	cacheHitCount     int64
	cacheMissCount    int64
	lastReset         time.Time
	mutex             sync.RWMutex
	slowQueryThreshold time.Duration
}

// NewPostgresAdvancedManager creates an advanced PostgreSQL manager
func NewPostgresAdvancedManager(baseManager *PostgresManager, config *AdvancedDatabaseConfig) *PostgresAdvancedManager {
	return &PostgresAdvancedManager{
		PostgresManager:  baseManager,
		advancedConfig:   config,
		queryCache:       NewQueryCache(config.QueryCacheSize, config.QueryCacheTTL),
		metricsCollector: NewMetricsCollector(),
	}
}

// InitializeAdvanced initializes advanced PostgreSQL features
func (dm *PostgresAdvancedManager) InitializeAdvanced() error {
	log.Printf("Initializing advanced PostgreSQL features...")
	
	// Setup connection pooling
	if err := dm.setupConnectionPooling(); err != nil {
		return fmt.Errorf("failed to setup connection pooling: %w", err)
	}
	
	// Setup read replicas if enabled
	if dm.advancedConfig.ReadReplicaEnabled {
		if err := dm.setupReadReplicas(); err != nil {
			return fmt.Errorf("failed to setup read replicas: %w", err)
		}
	}
	
	// Configure advanced PostgreSQL settings
	if err := dm.configureAdvancedSettings(); err != nil {
		return fmt.Errorf("failed to configure advanced settings: %w", err)
	}
	
	// Setup monitoring
	if dm.advancedConfig.MetricsEnabled {
		if err := dm.setupMetricsCollection(); err != nil {
			return fmt.Errorf("failed to setup metrics collection: %w", err)
		}
	}
	
	log.Printf("Advanced PostgreSQL features initialized successfully")
	return nil
}

// setupConnectionPooling configures advanced connection pooling
func (dm *PostgresAdvancedManager) setupConnectionPooling() error {
	log.Printf("Setting up connection pooling...")
	
	// Configure primary database connection pool
	primaryDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		dm.config.Host, dm.config.Port, dm.config.User, dm.config.Password, dm.config.DBName)
	
	primaryDB, err := sql.Open("postgres", primaryDSN)
	if err != nil {
		return fmt.Errorf("failed to open primary database: %w", err)
	}
	
	// Configure connection pool
	primaryDB.SetMaxOpenConns(dm.advancedConfig.MaxConnections)
	primaryDB.SetMaxIdleConns(dm.advancedConfig.MinConnections)
	primaryDB.SetConnMaxLifetime(dm.advancedConfig.MaxLifetime)
	primaryDB.SetConnMaxIdleTime(dm.advancedConfig.IdleTimeout)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), dm.advancedConfig.ConnectionTimeout)
	defer cancel()
	
	if err := primaryDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping primary database: %w", err)
	}
	
	dm.primaryDB = primaryDB
	
	log.Printf("Primary database connection pooling configured")
	return nil
}

// setupReadReplicas configures read replica connections
func (dm *PostgresAdvancedManager) setupReadReplicas() error {
	log.Printf("Setting up read replica connections...")
	
	replicaDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		dm.advancedConfig.ReadReplicaHost, dm.advancedConfig.ReadReplicaPort,
		dm.config.User, dm.config.Password, dm.config.DBName)
	
	replicaDB, err := sql.Open("postgres", replicaDSN)
	if err != nil {
		return fmt.Errorf("failed to open read replica database: %w", err)
	}
	
	// Configure replica connection pool
	replicaDB.SetMaxOpenConns(dm.advancedConfig.MaxConnections / 2) // Use half for read replicas
	replicaDB.SetMaxIdleConns(dm.advancedConfig.MinConnections / 2)
	replicaDB.SetConnMaxLifetime(dm.advancedConfig.MaxLifetime)
	replicaDB.SetConnMaxIdleTime(dm.advancedConfig.IdleTimeout)
	
	// Test replica connection
	ctx, cancel := context.WithTimeout(context.Background(), dm.advancedConfig.ConnectionTimeout)
	defer cancel()
	
	if err := replicaDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping read replica database: %w", err)
	}
	
	dm.readReplicaDB = replicaDB
	
	log.Printf("Read replica connection pooling configured")
	return nil
}

// configureAdvancedSettings configures advanced PostgreSQL settings
func (dm *PostgresAdvancedManager) configureAdvancedSettings() error {
	log.Printf("Configuring advanced PostgreSQL settings...")
	
	settings := []string{
		fmt.Sprintf("SET statement_timeout = '%dms'", dm.advancedConfig.StatementTimeout.Milliseconds()),
		fmt.Sprintf("SET lock_timeout = '%dms'", dm.advancedConfig.LockTimeout.Milliseconds()),
		fmt.Sprintf("SET idle_in_transaction_session_timeout = '%dms'", dm.advancedConfig.IdleInTransactionTimeout.Milliseconds()),
		"SET application_name = 'helixflow-postgres-advanced'",
	}
	
	for _, setting := range settings {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if _, err := dm.primaryDB.ExecContext(ctx, setting); err != nil {
			cancel()
			return fmt.Errorf("failed to set %s: %w", setting, err)
		}
		cancel()
	}
	
	// Configure PostgreSQL-specific settings
	pgSettings := []string{
		"SET work_mem = '256MB'",
		"SET maintenance_work_mem = '512MB'",
		"SET effective_cache_size = '2GB'",
		"SET shared_preload_libraries = 'pg_stat_statements'",
	}
	
	for _, setting := range pgSettings {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if _, err := dm.primaryDB.ExecContext(ctx, setting); err != nil {
			log.Printf("Warning: failed to set PostgreSQL setting %s: %v", setting, err)
		}
		cancel()
	}
	
	log.Printf("Advanced PostgreSQL settings configured")
	return nil
}

// setupMetricsCollection configures performance metrics collection
func (dm *PostgresAdvancedManager) setupMetricsCollection() error {
	log.Printf("Setting up metrics collection...")
	
	// Enable pg_stat_statements if available
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := dm.primaryDB.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	if err != nil {
		log.Printf("Warning: failed to enable pg_stat_statements: %v", err)
	}
	
	// Create metrics tables
	metricsTables := []string{
		`CREATE TABLE IF NOT EXISTS query_metrics (
			id SERIAL PRIMARY KEY,
			query_hash VARCHAR(64),
			query_text TEXT,
			execution_count BIGINT,
			total_time_ms BIGINT,
			mean_time_ms FLOAT8,
			stddev_time_ms FLOAT8,
			min_time_ms BIGINT,
			max_time_ms BIGINT,
			rows_returned BIGINT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS connection_metrics (
			id SERIAL PRIMARY KEY,
			active_connections INT,
			idle_connections INT,
			total_connections INT,
			wait_time_ms BIGINT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	
	for _, table := range metricsTables {
		if _, err := dm.primaryDB.ExecContext(ctx, table); err != nil {
			return fmt.Errorf("failed to create metrics table: %w", err)
		}
	}
	
	log.Printf("Metrics collection setup completed")
	return nil
}

// ExecuteQueryWithCache executes a query with caching support
func (dm *PostgresAdvancedManager) ExecuteQueryWithCache(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	startTime := time.Now()
	
	// Generate cache key
	cacheKey := dm.generateCacheKey(query, args)
	
	// Check cache first
	if dm.advancedConfig.QueryCacheEnabled {
		if cached, found := dm.queryCache.Get(cacheKey); found {
			dm.metricsCollector.RecordCacheHit()
			return cached, nil
		}
	}
	
	// Execute query
	var result interface{}
	err := dm.executeQuery(ctx, query, args, &result)
	
	// Record metrics
	executionTime := time.Since(startTime)
	dm.metricsCollector.RecordQuery(executionTime, err != nil)
	
	if err != nil {
		return nil, err
	}
	
	// Cache result
	if dm.advancedConfig.QueryCacheEnabled {
		dm.queryCache.Set(cacheKey, result)
		dm.metricsCollector.RecordCacheMiss()
	}
	
	// Check for slow queries
	if executionTime > dm.advancedConfig.SlowQueryThreshold {
		dm.metricsCollector.RecordSlowQuery(executionTime)
		log.Printf("Slow query detected: %v - %s", executionTime, query)
	}
	
	return result, nil
}

// executeQuery executes a query on the appropriate database (primary or replica)
func (dm *PostgresAdvancedManager) executeQuery(ctx context.Context, query string, args []interface{}, result interface{}) error {
	// Simple heuristic: SELECT queries can go to read replica
	isSelect := len(query) > 6 && query[:6] == "SELECT"
	
	var db *sql.DB
	if isSelect && dm.readReplicaDB != nil {
		db = dm.readReplicaDB
	} else {
		db = dm.primaryDB
	}
	
	return db.QueryRowContext(ctx, query, args...).Scan(result)
}

// GetMetrics returns current database performance metrics
func (dm *PostgresAdvancedManager) GetMetrics() map[string]interface{} {
	return dm.metricsCollector.GetMetrics()
}

// HealthCheck performs advanced health check
func (dm *PostgresAdvancedManager) HealthCheck() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Check primary database
	if err := dm.primaryDB.PingContext(ctx); err != nil {
		return "unhealthy", fmt.Errorf("primary database ping failed: %w", err)
	}
	
	// Check read replica if enabled
	if dm.readReplicaDB != nil {
		if err := dm.readReplicaDB.PingContext(ctx); err != nil {
			return "degraded", fmt.Errorf("read replica ping failed: %w", err)
		}
	}
	
	// Check connection pool health (simplified for now)
	// stats := dm.connectionPool.GetStats()
	// if stats.ActiveConnections > dm.advancedConfig.MaxConnections*90/100 {
	//     return "degraded", fmt.Errorf("connection pool at 90% capacity")
	// }
	
	return "healthy", nil
}

// Helper functions

func NewQueryCache(size int, ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: size,
		ttl:     ttl,
	}
}

func (qc *QueryCache) Get(key string) (interface{}, bool) {
	qc.mutex.RLock()
	defer qc.mutex.RUnlock()
	
	entry, exists := qc.cache[key]
	if !exists {
		return nil, false
	}
	
	// Check if entry is expired
	if time.Since(entry.Timestamp) > qc.ttl {
		return nil, false
	}
	
	return entry.Data, true
}

func (qc *QueryCache) Set(key string, data interface{}) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()
	
	// Evict old entries if cache is full
	if len(qc.cache) >= qc.maxSize {
		qc.evictOldest()
	}
	
	qc.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
}

func (qc *QueryCache) evictOldest() {
	// Simple LRU eviction - remove oldest entry
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range qc.cache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}
	
	if oldestKey != "" {
		delete(qc.cache, oldestKey)
	}
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		lastReset:          time.Now(),
		slowQueryThreshold: 100 * time.Millisecond, // Default threshold
	}
}

func (mc *MetricsCollector) RecordQuery(duration time.Duration, isError bool) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	mc.queryCount++
	if isError {
		mc.errorCount++
	}
	if duration > mc.slowQueryThreshold {
		mc.slowQueryCount++
	}
}

func (mc *MetricsCollector) RecordCacheHit() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.cacheHitCount++
}

func (mc *MetricsCollector) RecordCacheMiss() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.cacheMissCount++
}

func (mc *MetricsCollector) RecordSlowQuery(duration time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.slowQueryCount++
}

func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	return map[string]interface{}{
		"query_count":        mc.queryCount,
		"error_count":        mc.errorCount,
		"slow_query_count":   mc.slowQueryCount,
		"cache_hit_count":    mc.cacheHitCount,
		"cache_miss_count":   mc.cacheMissCount,
		"cache_hit_ratio":    float64(mc.cacheHitCount) / float64(mc.cacheHitCount+mc.cacheMissCount) * 100,
		"error_rate":         float64(mc.errorCount) / float64(mc.queryCount) * 100,
		"last_reset":         mc.lastReset.Format(time.RFC3339),
	}
}

func (dm *PostgresAdvancedManager) generateCacheKey(query string, args []interface{}) string {
	// Simple cache key generation - in production, use proper hashing
	return fmt.Sprintf("%s:%v", query, args)
}