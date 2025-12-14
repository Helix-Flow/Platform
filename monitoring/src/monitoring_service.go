package main

import (
	"context"
	"fmt"
)

// Helper functions for monitoring service

// RecordAPIMetrics logs API metrics (helper function)
func RecordAPIMetrics(ctx context.Context, userID, method, path string, statusCode int, latencyMs int64) error {
	// Mock implementation - log the metrics
	fmt.Printf("[METRICS] User: %s, Method: %s, Path: %s, Status: %d, Latency: %dms\n", 
		userID, method, path, statusCode, latencyMs)
	return nil
}

// RecordInferenceMetrics logs inference metrics (helper function)
func RecordInferenceMetrics(modelID, userID string, latencyMs int64, tokenCount int32) error {
	fmt.Printf("[INFERENCE_METRICS] Model: %s, User: %s, Latency: %dms, Tokens: %d\n", 
		modelID, userID, latencyMs, tokenCount)
	return nil
}

// LogAPIRequest logs API request (helper function)
func LogAPIRequest(userID, method, path string, statusCode int, latencyMs int64, ipAddress string) {
	fmt.Printf("[API_LOG] User: %s, Method: %s, Path: %s, Status: %d, Latency: %dms, IP: %s\n", 
		userID, method, path, statusCode, latencyMs, ipAddress)
}