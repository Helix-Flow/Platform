package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Incident struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
}

type IncidentStore struct {
	mu        sync.RWMutex
	incidents []Incident
}

func NewIncidentStore() *IncidentStore {
	return &IncidentStore{
		incidents: make([]Incident, 0),
	}
}

func (s *IncidentStore) Add(incident Incident) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.incidents = append(s.incidents, incident)
}

func (s *IncidentStore) List() []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Incident, len(s.incidents))
	copy(result, s.incidents)
	return result
}

func (s *IncidentStore) CountByType(incidentType string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, i := range s.incidents {
		if i.Type == incidentType {
			count++
		}
	}
	return count
}

var incidentStore = NewIncidentStore()

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

	// Security incidents endpoint
	mux.HandleFunc("/api/security/incidents", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "GET":
			incidents := incidentStore.List()
			json.NewEncoder(w).Encode(incidents)
		case "POST":
			var incident Incident
			if err := json.NewDecoder(r.Body).Decode(&incident); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if incident.ID == "" {
				incident.ID = fmt.Sprintf("incident-%d", time.Now().UnixNano())
			}
			if incident.Timestamp.IsZero() {
				incident.Timestamp = time.Now()
			}
			incidentStore.Add(incident)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(incident)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Metrics endpoints
	mux.HandleFunc("/api/metrics/cpu", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"cpu_percent": 45.2, "cores": 4})
	})
	mux.HandleFunc("/api/metrics/memory", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"memory_percent": 67.8, "total_gb": 16})
	})
	mux.HandleFunc("/api/metrics/disk", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"disk_percent": 23.1, "total_gb": 500})
	})
	mux.HandleFunc("/api/metrics/gpu", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"gpus": []map[string]interface{}{
				{"utilization": 45, "memory_usage": 60},
			},
		})
	})

	// Alerts endpoint
	mux.HandleFunc("/api/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "high_cpu", "status": "resolved", "severity": "warning"},
		})
	})

	// Compliance endpoints
	mux.HandleFunc("/api/compliance/report", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"gdpr_compliance":  true,
			"soc2_compliance":  true,
			"last_audit":       time.Now().Format(time.RFC3339),
			"violations":       []string{},
		})
	})
	mux.HandleFunc("/api/compliance/checks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"check_name": "encryption_at_rest", "status": "pass", "last_run": time.Now().Format(time.RFC3339)},
			{"check_name": "access_controls", "status": "pass", "last_run": time.Now().Format(time.RFC3339)},
		})
	})

	// Audit logs endpoint
	mux.HandleFunc("/api/audit/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"timestamp": time.Now().Format(time.RFC3339), "user_id": "admin", "action": "login", "resource": "auth", "ip_address": "127.0.0.1"},
		})
	})

	// Get port from environment
	port := getEnv("MONITORING_PORT", "8083")
	
	log.Printf("Monitoring service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start monitoring service: %v", err)
	}
}

// ReportIncident sends an incident to the monitoring service
func ReportIncident(incident Incident) {
	monitoringURL := getEnv("MONITORING_SERVICE_URL", "http://localhost:8083")
	data, _ := json.Marshal(incident)
	resp, err := http.Post(monitoringURL+"/api/security/incidents", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to report incident: %v", err)
		return
	}
	defer resp.Body.Close()
}
