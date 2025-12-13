package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type MonitoringService struct {
	redisClient         *redis.Client
	prometheusAPI       api.Client
	registry            *prometheus.Registry
	requestCount        *prometheus.CounterVec
	requestLatency      *prometheus.HistogramVec
	gpuMemoryUsage      *prometheus.GaugeVec
	gpuUtilization      *prometheus.GaugeVec
	activeConnections   *prometheus.GaugeVec
	queueSize           *prometheus.GaugeVec
	modelInferenceCount *prometheus.CounterVec
	errorCount          *prometheus.CounterVec
}

func NewMonitoringService() *MonitoringService {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_HOST", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	prometheusClient, err := api.NewClient(api.Config{
		Address: getEnv("PROMETHEUS_URL", "http://localhost:9090"),
	})
	if err != nil {
		log.Printf("Failed to create Prometheus client: %v", err)
	}

	registry := prometheus.NewRegistry()

	// Define metrics
	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helixflow_requests_total",
			Help: "Total number of requests",
		},
		[]string{"service", "endpoint", "method", "status"},
	)

	requestLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "helixflow_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "endpoint"},
	)

	gpuMemoryUsage := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helixflow_gpu_memory_usage_bytes",
			Help: "GPU memory usage in bytes",
		},
		[]string{"gpu_id"},
	)

	gpuUtilization := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helixflow_gpu_utilization_percent",
			Help: "GPU utilization percentage",
		},
		[]string{"gpu_id"},
	)

	activeConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helixflow_active_connections",
			Help: "Number of active connections",
		},
		[]string{"service"},
	)

	queueSize := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helixflow_queue_size",
			Help: "Current queue size",
		},
		[]string{"queue_name"},
	)

	modelInferenceCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helixflow_model_inferences_total",
			Help: "Total number of model inferences",
		},
		[]string{"model_name", "gpu_id"},
	)

	errorCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helixflow_errors_total",
			Help: "Total number of errors",
		},
		[]string{"service", "error_type"},
	)

	// Register metrics
	registry.MustRegister(
		requestCount,
		requestLatency,
		gpuMemoryUsage,
		gpuUtilization,
		activeConnections,
		queueSize,
		modelInferenceCount,
		errorCount,
	)

	return &MonitoringService{
		redisClient:         redisClient,
		prometheusAPI:       prometheusClient,
		registry:            registry,
		requestCount:        requestCount,
		requestLatency:      requestLatency,
		gpuMemoryUsage:      gpuMemoryUsage,
		gpuUtilization:      gpuUtilization,
		activeConnections:   activeConnections,
		queueSize:           queueSize,
		modelInferenceCount: modelInferenceCount,
		errorCount:          errorCount,
	}
}

func (ms *MonitoringService) RecordRequest(service, endpoint, method, status string, duration float64) {
	ms.requestCount.WithLabelValues(service, endpoint, method, status).Inc()
	ms.requestLatency.WithLabelValues(service, endpoint).Observe(duration)
}

func (ms *MonitoringService) RecordInference(modelName, gpuID string) {
	ms.modelInferenceCount.WithLabelValues(modelName, gpuID).Inc()
}

func (ms *MonitoringService) RecordError(service, errorType string) {
	ms.errorCount.WithLabelValues(service, errorType).Inc()
}

func (ms *MonitoringService) UpdateSystemMetrics() {
	// CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		// Could add CPU gauge metric here
		log.Printf("CPU usage: %.2f%%", cpuPercent[0])
	}

	// Memory usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		log.Printf("Memory usage: %.2f%%", memInfo.UsedPercent)
	}

	// Disk usage
	diskInfo, err := disk.Usage("/")
	if err == nil {
		log.Printf("Disk usage: %.2f%%", diskInfo.UsedPercent)
	}
}

func (ms *MonitoringService) UpdateGPUMetrics() {
	// Placeholder for GPU metrics - would integrate with NVIDIA/AMD libraries
	// For now, simulate some GPU metrics
	gpuIDs := []string{"gpu_0", "gpu_1", "gpu_2", "gpu_3"}
	for _, gpuID := range gpuIDs {
		// Simulate GPU memory usage (in bytes)
		ms.gpuMemoryUsage.WithLabelValues(gpuID).Set(float64(8 * 1024 * 1024 * 1024)) // 8GB
		// Simulate GPU utilization
		ms.gpuUtilization.WithLabelValues(gpuID).Set(75.5)
	}
}

func (ms *MonitoringService) UpdateQueueMetrics() {
	// Check Redis queues
	queues := []string{"inference_queue", "api_requests", "background_jobs"}

	for _, queue := range queues {
		size, err := ms.redisClient.LLen(context.Background(), queue).Result()
		if err == nil {
			ms.queueSize.WithLabelValues(queue).Set(float64(size))
		}
	}
}

func (ms *MonitoringService) GetAlerts() map[string]interface{} {
	// In a real implementation, this would query Prometheus Alertmanager
	alerts := []map[string]interface{}{
		{
			"alertname":   "HighGPUUtilization",
			"severity":    "warning",
			"description": "GPU utilization above 90%",
			"state":       "firing",
		},
		{
			"alertname":   "HighMemoryUsage",
			"severity":    "critical",
			"description": "Memory usage above 95%",
			"state":       "firing",
		},
	}

	return map[string]interface{}{
		"alerts": alerts,
	}
}

func (ms *MonitoringService) GetMetrics() map[string]interface{} {
	// Collect current metrics
	ms.UpdateSystemMetrics()
	ms.UpdateGPUMetrics()
	ms.UpdateQueueMetrics()

	// Get CPU info
	cpuPercent, _ := cpu.Percent(time.Second, false)
	memInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")

	return map[string]interface{}{
		"cpu_percent":    cpuPercent,
		"memory_percent": memInfo.UsedPercent,
		"disk_percent":   diskInfo.UsedPercent,
		"memory_used":    memInfo.Used,
		"memory_total":   memInfo.Total,
		"disk_used":      diskInfo.Used,
		"disk_total":     diskInfo.Total,
		"timestamp":      time.Now().Unix(),
	}
}

// HTTP Handlers
func (ms *MonitoringService) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "monitoring",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ms *MonitoringService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Update metrics
	ms.UpdateSystemMetrics()
	ms.UpdateGPUMetrics()
	ms.UpdateQueueMetrics()

	// Serve Prometheus metrics
	promhttp.HandlerFor(ms.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

func (ms *MonitoringService) alertsHandler(w http.ResponseWriter, r *http.Request) {
	alerts := ms.GetAlerts()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (ms *MonitoringService) recordRequestHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Service  string  `json:"service"`
		Endpoint string  `json:"endpoint"`
		Method   string  `json:"method"`
		Status   string  `json:"status"`
		Duration float64 `json:"duration"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ms.RecordRequest(req.Service, req.Endpoint, req.Method, req.Status, req.Duration)
	w.WriteHeader(http.StatusOK)
}

func (ms *MonitoringService) recordInferenceHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelName string `json:"model_name"`
		GPUID     string `json:"gpu_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ms.RecordInference(req.ModelName, req.GPUID)
	w.WriteHeader(http.StatusOK)
}

func (ms *MonitoringService) recordErrorHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Service   string `json:"service"`
		ErrorType string `json:"error_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ms.RecordError(req.Service, req.ErrorType)
	w.WriteHeader(http.StatusOK)
}

func (ms *MonitoringService) getMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := ms.GetMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	monitoringService := NewMonitoringService()

	r := mux.NewRouter()

	// Routes
	r.HandleFunc("/health", monitoringService.healthHandler).Methods("GET")
	r.HandleFunc("/metrics", monitoringService.metricsHandler).Methods("GET")
	r.HandleFunc("/alerts", monitoringService.alertsHandler).Methods("GET")
	r.HandleFunc("/api/metrics", monitoringService.getMetricsHandler).Methods("GET")
	r.HandleFunc("/api/metrics/request", monitoringService.recordRequestHandler).Methods("POST")
	r.HandleFunc("/api/metrics/inference", monitoringService.recordInferenceHandler).Methods("POST")
	r.HandleFunc("/api/metrics/error", monitoringService.recordErrorHandler).Methods("POST")

	// Start server
	port := getEnv("PORT", "8081")
	log.Printf("Starting monitoring service on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
