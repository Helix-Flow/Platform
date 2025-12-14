package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Start gRPC server in a goroutine
	go func() {
		if err := StartGRPCServer(); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Create HTTP server for health checks and metrics
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status := "healthy" // Simple health check
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "%s", "timestamp": "%s"}`, status, time.Now().Format(time.RFC3339))
	})
	
	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := map[string]interface{}{
			"cpu_usage":    45.2,
			"memory_usage": 67.8,
			"disk_usage":   23.1,
			"uptime":       "24h",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})

	// Get port from environment
	port := getEnv("MONITORING_PORT", "8083")
	
	log.Printf("Monitoring service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start monitoring service: %v", err)
	}
}