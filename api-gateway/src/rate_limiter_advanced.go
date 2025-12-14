package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// AdvancedRateLimiter implements sophisticated rate limiting with multiple algorithms
type AdvancedRateLimiter struct {
	redisClient *redis.Client
	
	// Rate limiting algorithms
	tokenBucket    *TokenBucketLimiter
	slidingWindow  *SlidingWindowLimiter
	fixedWindow    *FixedWindowLimiter
	
	// Configuration
	config *RateLimitConfig
	
	// Caching
	responseCache *ResponseCache
	
	// Metrics
	metrics *RateLimitMetrics
}

// RateLimitConfig holds advanced rate limiting configuration
type RateLimitConfig struct {
	// General settings
	Enabled              bool          `json:"enabled"`
	RedisEnabled         bool          `json:"redis_enabled"`
	DefaultRateLimit     int           `json:"default_rate_limit"`
	DefaultWindow        time.Duration `json:"default_window"`
	
	// Algorithm selection
	PrimaryAlgorithm     string        `json:"primary_algorithm"` // "token_bucket", "sliding_window", "fixed_window"
	FallbackAlgorithm    string        `json:"fallback_algorithm"`
	
	// Token bucket settings
	TokenBucketCapacity  int           `json:"token_bucket_capacity"`
	TokenRefillRate      int           `json:"token_refill_rate"`
	TokenRefillInterval  time.Duration `json:"token_refill_interval"`
	
	// Sliding window settings
	SlidingWindowSize    int           `json:"sliding_window_size"`
	SlidingWindowGranularity time.Duration `json:"sliding_window_granularity"`
	
	// Fixed window settings
	FixedWindowSize      time.Duration `json:"fixed_window_size"`
	
	// Caching settings
	CacheEnabled         bool          `json:"cache_enabled"`
	CacheTTL             time.Duration `json:"cache_ttl"`
	CacheMaxSize         int           `json:"cache_max_size"`
	
	// Advanced features
	BurstEnabled         bool          `json:"burst_enabled"`
	BurstCapacity        int           `json:"burst_capacity"`
	BurstDuration        time.Duration `json:"burst_duration"`
	
	// Monitoring
	MetricsEnabled       bool          `json:"metrics_enabled"`
	SlowRequestThreshold time.Duration `json:"slow_request_threshold"`
}

// TokenBucketLimiter implements token bucket algorithm
type TokenBucketLimiter struct {
	capacity      int
	refillRate    int
	refillInterval time.Duration
	
	// Per-user tracking
	buckets map[string]*TokenBucket
	mutex   sync.RWMutex
}

// TokenBucket represents a user's token bucket
type TokenBucket struct {
	tokens       int
	lastRefill   time.Time
	capacity     int
	refillRate   int
}

// SlidingWindowLimiter implements sliding window algorithm
type SlidingWindowLimiter struct {
	windowSize    time.Duration
	granularity   time.Duration
	maxRequests   int
	
	// Per-user tracking
	windows map[string]*SlidingWindow
	mutex   sync.RWMutex
}

// SlidingWindow represents a user's sliding window
type SlidingWindow struct {
	counts    map[int64]int // timestamp -> count
	windowStart int64
	windowEnd   int64
	totalCount  int
}

// FixedWindowLimiter implements fixed window algorithm
type FixedWindowLimiter struct {
	windowSize  time.Duration
	maxRequests int
	
	// Per-user tracking
	windows map[string]*FixedWindow
	mutex   sync.RWMutex
}

// FixedWindow represents a user's fixed window
type FixedWindow struct {
	count      int
	windowStart time.Time
	windowEnd   time.Time
}

// ResponseCache provides intelligent response caching
type ResponseCache struct {
	redisClient *redis.Client
	localCache  map[string]*CacheEntry
	mutex       sync.RWMutex
	maxSize     int
	ttl         time.Duration
}

// CacheEntry represents a cached response
type CacheEntry struct {
	Data      []byte
	Headers   map[string]string
	StatusCode int
	Timestamp time.Time
}

// RateLimitMetrics tracks rate limiting performance
type RateLimitMetrics struct {
	requestsTotal      int64
	requestsAllowed    int64
	requestsDenied     int64
	cacheHits          int64
	cacheMisses        int64
	avgResponseTime    time.Duration
	slowRequests       int64
	lastReset          time.Time
	mutex              sync.RWMutex
}

// NewAdvancedRateLimiter creates an advanced rate limiter
func NewAdvancedRateLimiter(redisClient *redis.Client, config *RateLimitConfig) *AdvancedRateLimiter {
	rl := &AdvancedRateLimiter{
		redisClient:   redisClient,
		config:        config,
		metrics:       NewRateLimitMetrics(),
	}
	
	// Initialize rate limiting algorithms
	rl.tokenBucket = NewTokenBucketLimiter(
		config.TokenBucketCapacity,
		config.TokenRefillRate,
		config.TokenRefillInterval,
	)
	
	rl.slidingWindow = NewSlidingWindowLimiter(
		config.SlidingWindowSize,
		config.SlidingWindowGranularity,
		config.DefaultRateLimit,
	)
	
	rl.fixedWindow = NewFixedWindowLimiter(
		config.FixedWindowSize,
		config.DefaultRateLimit,
	)
	
	// Initialize response cache
	if config.CacheEnabled {
		rl.responseCache = NewResponseCache(redisClient, config.CacheMaxSize, config.CacheTTL)
	}
	
	return rl
}

// CheckRateLimitAdvanced checks rate limit using advanced algorithms
func (rl *AdvancedRateLimiter) CheckRateLimitAdvanced(ctx context.Context, userID string, requestType string) (bool, map[string]interface{}) {
	startTime := time.Now()
	
	// Record request
	rl.metrics.RecordRequest()
	
	// Determine which algorithm to use
	var allowed bool
	var details map[string]interface{}
	
	switch rl.config.PrimaryAlgorithm {
	case "token_bucket":
		allowed, details = rl.tokenBucket.Check(userID, requestType)
	case "sliding_window":
		allowed, details = rl.slidingWindow.Check(userID, requestType)
	case "fixed_window":
		allowed, details = rl.fixedWindow.Check(userID, requestType)
	default:
		allowed, details = rl.tokenBucket.Check(userID, requestType)
	}
	
	// If primary algorithm denies, try fallback
	if !allowed && rl.config.FallbackAlgorithm != "" {
		switch rl.config.FallbackAlgorithm {
		case "token_bucket":
			allowed, details = rl.tokenBucket.Check(userID, requestType)
		case "sliding_window":
			allowed, details = rl.slidingWindow.Check(userID, requestType)
		case "fixed_window":
			allowed, details = rl.fixedWindow.Check(userID, requestType)
		}
	}
	
	// Record result
	if allowed {
		rl.metrics.RecordAllowed()
	} else {
		rl.metrics.RecordDenied()
	}
	
	// Check for slow requests
	responseTime := time.Since(startTime)
	if responseTime > rl.config.SlowRequestThreshold {
		rl.metrics.RecordSlowRequest(responseTime)
	}
	
	// Add metrics to details
	details["response_time_ms"] = responseTime.Milliseconds()
	details["algorithm_used"] = rl.config.PrimaryAlgorithm
	if !allowed {
		details["fallback_used"] = rl.config.FallbackAlgorithm
	}
	
	return allowed, details
}

// CacheResponse caches a response for future requests
func (rl *AdvancedRateLimiter) CacheResponse(ctx context.Context, key string, response []byte, headers map[string]string, statusCode int) error {
	if !rl.config.CacheEnabled {
		return nil
	}
	
	return rl.responseCache.Set(ctx, key, response, headers, statusCode)
}

// GetCachedResponse retrieves a cached response
func (rl *AdvancedRateLimiter) GetCachedResponse(ctx context.Context, key string) ([]byte, map[string]string, int, bool) {
	if !rl.config.CacheEnabled {
		return nil, nil, 0, false
	}
	
	return rl.responseCache.Get(ctx, key)
}

// GetMetrics returns rate limiting metrics
func (rl *AdvancedRateLimiter) GetMetrics() map[string]interface{} {
	return rl.metrics.GetMetrics()
}

// Token Bucket Algorithm Implementation

func NewTokenBucketLimiter(capacity, refillRate int, refillInterval time.Duration) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:       capacity,
		refillRate:     refillRate,
		refillInterval: refillInterval,
		buckets:        make(map[string]*TokenBucket),
	}
}

func (tbl *TokenBucketLimiter) Check(userID string, requestType string) (bool, map[string]interface{}) {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()
	
	bucket, exists := tbl.buckets[userID]
	if !exists {
		bucket = &TokenBucket{
			tokens:     tbl.capacity,
			lastRefill: time.Now(),
			capacity:   tbl.capacity,
			refillRate: tbl.refillRate,
		}
		tbl.buckets[userID] = bucket
	}
	
	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / tbl.refillInterval) * bucket.refillRate
	
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.tokens+tokensToAdd, bucket.capacity)
		bucket.lastRefill = now
	}
	
	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true, map[string]interface{}{
			"tokens_remaining": bucket.tokens,
			"tokens_capacity":  bucket.capacity,
			"algorithm":        "token_bucket",
		}
	}
	
	return false, map[string]interface{}{
			"tokens_remaining": bucket.tokens,
			"tokens_capacity":  bucket.capacity,
			"algorithm":        "token_bucket",
			"retry_after":      tbl.refillInterval.Seconds(),
	}
}

// Sliding Window Algorithm Implementation

func NewSlidingWindowLimiter(windowSize int, granularity time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  time.Duration(windowSize) * time.Second,
		granularity: granularity,
		maxRequests: maxRequests,
		windows:     make(map[string]*SlidingWindow),
	}
}

func (swl *SlidingWindowLimiter) Check(userID string, requestType string) (bool, map[string]interface{}) {
	swl.mutex.Lock()
	defer swl.mutex.Unlock()
	
	window, exists := swl.windows[userID]
	if !exists {
		window = &SlidingWindow{
			counts:      make(map[int64]int),
			windowStart: time.Now().Unix(),
			windowEnd:   time.Now().Add(swl.windowSize).Unix(),
			totalCount:  0,
		}
		swl.windows[userID] = window
	}
	
	now := time.Now()
	currentSlot := now.Unix() / int64(swl.granularity.Seconds())
	
	// Clean up old slots
	cutoff := now.Add(-swl.windowSize).Unix() / int64(swl.granularity.Seconds())
	for slot, count := range window.counts {
		if slot < cutoff {
			window.totalCount -= count
			delete(window.counts, slot)
		}
	}
	
	// Update current slot
	currentCount := window.counts[currentSlot]
	if window.totalCount-currentCount+1 <= swl.maxRequests {
		window.counts[currentSlot] = currentCount + 1
		window.totalCount = window.totalCount - currentCount + (currentCount + 1)
		
		return true, map[string]interface{}{
			"requests_in_window": window.totalCount,
			"max_requests":       swl.maxRequests,
			"algorithm":          "sliding_window",
		}
	}
	
	return false, map[string]interface{}{
		"requests_in_window": window.totalCount,
		"max_requests":       swl.maxRequests,
		"algorithm":          "sliding_window",
		"window_size":        swl.windowSize.Seconds(),
	}
}

// Fixed Window Algorithm Implementation

func NewFixedWindowLimiter(windowSize time.Duration, maxRequests int) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		windows:     make(map[string]*FixedWindow),
	}
}

func (fwl *FixedWindowLimiter) Check(userID string, requestType string) (bool, map[string]interface{}) {
	fwl.mutex.Lock()
	defer fwl.mutex.Unlock()
	
	now := time.Now()
	window, exists := fwl.windows[userID]
	
	// Create new window if doesn't exist or current window has expired
	if !exists || now.After(window.windowEnd) {
		window = &FixedWindow{
			count:       0,
			windowStart: now,
			windowEnd:   now.Add(fwl.windowSize),
		}
		fwl.windows[userID] = window
	}
	
	// Check if we can accept more requests
	if window.count < fwl.maxRequests {
		window.count++
		return true, map[string]interface{}{
			"requests_in_window": window.count,
			"max_requests":       fwl.maxRequests,
			"algorithm":          "fixed_window",
		}
	}
	
	return false, map[string]interface{}{
		"requests_in_window": window.count,
		"max_requests":       fwl.maxRequests,
		"algorithm":          "fixed_window",
		"window_end":         window.windowEnd.Unix(),
	}
}

// Response Cache Implementation

func NewResponseCache(redisClient *redis.Client, maxSize int, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		redisClient: redisClient,
		localCache:  make(map[string]*CacheEntry),
		maxSize:     maxSize,
		ttl:         ttl,
	}
}

func (rc *ResponseCache) Set(ctx context.Context, key string, response []byte, headers map[string]string, statusCode int) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	entry := &CacheEntry{
		Data:       response,
		Headers:    headers,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
	
	// Store in Redis for distributed caching
	entryData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}
	
	if rc.redisClient != nil {
		if err := rc.redisClient.Set(ctx, key, entryData, rc.ttl).Err(); err != nil {
			log.Printf("Warning: failed to cache in Redis: %v", err)
		}
	}
	
	// Store in local cache
	rc.localCache[key] = entry
	
	// Evict old entries if cache is full
	if len(rc.localCache) > rc.maxSize {
		rc.evictOldest()
	}
	
	return nil
}

func (rc *ResponseCache) Get(ctx context.Context, key string) ([]byte, map[string]string, int, bool) {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	
	// Check local cache first
	if entry, exists := rc.localCache[key]; exists {
		if time.Since(entry.Timestamp) < rc.ttl {
			return entry.Data, entry.Headers, entry.StatusCode, true
		}
	}
	
	// Check Redis cache
	if rc.redisClient != nil {
		cachedData, err := rc.redisClient.Get(ctx, key).Result()
		if err == nil {
			var entry CacheEntry
			if err := json.Unmarshal([]byte(cachedData), &entry); err == nil {
				if time.Since(entry.Timestamp) < rc.ttl {
					return entry.Data, entry.Headers, entry.StatusCode, true
				}
			}
		}
	}
	
	return nil, nil, 0, false
}

func (rc *ResponseCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range rc.localCache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}
	
	if oldestKey != "" {
		delete(rc.localCache, oldestKey)
	}
}

// Rate Limit Metrics Implementation

func NewRateLimitMetrics() *RateLimitMetrics {
	return &RateLimitMetrics{
		lastReset: time.Now(),
	}
}

func (rlm *RateLimitMetrics) RecordRequest() {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.requestsTotal++
}

func (rlm *RateLimitMetrics) RecordAllowed() {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.requestsAllowed++
}

func (rlm *RateLimitMetrics) RecordDenied() {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.requestsDenied++
}

func (rlm *RateLimitMetrics) RecordCacheHit() {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.cacheHits++
}

func (rlm *RateLimitMetrics) RecordCacheMiss() {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.cacheMisses++
}

func (rlm *RateLimitMetrics) RecordSlowRequest(duration time.Duration) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.slowRequests++
	rlm.avgResponseTime = (rlm.avgResponseTime + duration) / 2
}

func (rlm *RateLimitMetrics) GetMetrics() map[string]interface{} {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()
	
	totalRequests := rlm.requestsTotal
	if totalRequests == 0 {
		totalRequests = 1 // Avoid division by zero
	}
	
	return map[string]interface{}{
		"requests_total":       rlm.requestsTotal,
		"requests_allowed":     rlm.requestsAllowed,
		"requests_denied":      rlm.requestsDenied,
		"allow_rate":           float64(rlm.requestsAllowed) / float64(totalRequests) * 100,
		"deny_rate":            float64(rlm.requestsDenied) / float64(totalRequests) * 100,
		"cache_hit_count":      rlm.cacheHits,
		"cache_miss_count":     rlm.cacheMisses,
		"cache_hit_ratio":      float64(rlm.cacheHits) / float64(rlm.cacheHits+rlm.cacheMisses) * 100,
		"slow_requests":        rlm.slowRequests,
		"avg_response_time_ms": rlm.avgResponseTime.Milliseconds(),
		"last_reset":           rlm.lastReset.Format(time.RFC3339),
	}
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}