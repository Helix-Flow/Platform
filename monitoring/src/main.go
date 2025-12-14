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
	// Create monitoring service server
	monitoringServer := NewMonitoringServiceServer()

	// Create HTTP server for health checks
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status := monitoringServer.HealthCheck()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "%s", "timestamp": "%s"}`, status, time.Now().Format(time.RFC3339))
	})
	
	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := monitoringServer.GetSystemMetrics()
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