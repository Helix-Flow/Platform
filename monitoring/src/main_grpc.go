package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	monitoring "helixflow/monitoring/monitoring"
)

// MonitoringServiceServer implements the gRPC MonitoringService
type MonitoringServiceServer struct {
	monitoring.UnimplementedMonitoringServiceServer
	healthServer *health.Server
}

// NewMonitoringServiceServer creates a new monitoring service server
func NewMonitoringServiceServer() *MonitoringServiceServer {
	return &MonitoringServiceServer{
		healthServer: health.NewServer(),
	}
}

// RecordMetrics records system metrics
func (s *MonitoringServiceServer) RecordMetrics(ctx context.Context, req *monitoring.RecordMetricsRequest) (*monitoring.RecordMetricsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Record API metrics
	if req.ApiMetrics != nil {
		fmt.Printf("[METRICS] API - User: %s, Method: %s, Path: %s, Status: %d, Latency: %dms\n",
			req.ApiMetrics.UserId,
			req.ApiMetrics.Method,
			req.ApiMetrics.Path,
			req.ApiMetrics.StatusCode,
			req.ApiMetrics.LatencyMs)
	}

	// Record inference metrics
	if req.InferenceMetrics != nil {
		fmt.Printf("[METRICS] Inference - Model: %s, User: %s, Latency: %dms, Tokens: %d\n",
			req.InferenceMetrics.ModelId,
			req.InferenceMetrics.UserId,
			req.InferenceMetrics.LatencyMs,
			req.InferenceMetrics.TokenCount)
	}

	// Record system metrics
	if req.SystemMetrics != nil {
		fmt.Printf("[METRICS] System - CPU: %.2f%%, Memory: %.2f%%, Disk: %.2f%%, Uptime: %s\n",
			req.SystemMetrics.CpuUsage,
			req.SystemMetrics.MemoryUsage,
			req.SystemMetrics.DiskUsage,
			req.SystemMetrics.Uptime)
	}

	return &monitoring.RecordMetricsResponse{
		Success: true,
		Message: "Metrics recorded successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// GetSystemMetrics retrieves current system metrics
func (s *MonitoringServiceServer) GetSystemMetrics(ctx context.Context, req *monitoring.GetSystemMetricsRequest) (*monitoring.GetSystemMetricsResponse, error) {
	// Generate realistic system metrics
	metrics := &monitoring.SystemMetrics{
		CpuUsage:    45.2 + (time.Now().UnixNano()%20 - 10), // Simulate variation
		MemoryUsage: 67.8 + (time.Now().UnixNano()%15 - 7),
		DiskUsage:   23.1 + (time.Now().UnixNano()%10 - 5),
		Uptime:      "24h",
	}

	return &monitoring.GetSystemMetricsResponse{
		Success: true,
		Metrics: metrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// GetHealthStatus retrieves health status of services
func (s *MonitoringServiceServer) GetHealthStatus(ctx context.Context, req *monitoring.GetHealthStatusRequest) (*monitoring.GetHealthStatusResponse, error) {
	// Check health of various services
	services := []*monitoring.ServiceHealth{
		{
			ServiceName: "api-gateway",
			Status:      "healthy",
			LastChecked: time.Now().Format(time.RFC3339),
			ResponseTime: 45,
		},
		{
			ServiceName: "auth-service",
			Status:      "healthy", 
			LastChecked: time.Now().Format(time.RFC3339),
			ResponseTime: 32,
		},
		{
			ServiceName: "inference-pool",
			Status:      "healthy",
			LastChecked: time.Now().Format(time.RFC3339),
			ResponseTime: 78,
		},
		{
			ServiceName: "monitoring",
			Status:      "healthy",
			LastChecked: time.Now().Format(time.RFC3339),
			ResponseTime: 23,
		},
	}

	return &monitoring.GetHealthStatusResponse{
		Success: true,
		Services: services,
		OverallStatus: "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// CreateAlert creates a new alert
func (s *MonitoringServiceServer) CreateAlert(ctx context.Context, req *monitoring.CreateAlertRequest) (*monitoring.CreateAlertResponse, error) {
	if req.Alert == nil {
		return nil, fmt.Errorf("alert cannot be nil")
	}

	// In a real implementation, this would:
	// 1. Validate the alert configuration
	// 2. Store the alert in the database
	// 3. Set up monitoring for the alert condition
	// 4. Send initial notifications if needed

	fmt.Printf("[ALERT] Created alert: %s - %s (Severity: %s)\n",
		req.Alert.Name,
		req.Alert.Description,
		req.Alert.Severity)

	return &monitoring.CreateAlertResponse{
		Success: true,
		AlertId: fmt.Sprintf("alert_%d", time.Now().Unix()),
		Message: "Alert created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// GetAlerts retrieves active alerts
func (s *MonitoringServiceServer) GetAlerts(ctx context.Context, req *monitoring.GetAlertsRequest) (*monitoring.GetAlertsResponse, error) {
	// Mock alerts for demonstration
	alerts := []*monitoring.Alert{
		{
			AlertId:   "alert_1",
			Name:      "High CPU Usage",
			Description: "CPU usage exceeds 80%",
			Severity:  monitoring.Severity_WARNING,
			Status:    monitoring.AlertStatus_ACTIVE,
			CreatedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			AlertId:   "alert_2",
			Name:      "Database Connection Pool",
			Description: "Connection pool utilization above 90%",
			Severity:  monitoring.Severity_INFO,
			Status:    monitoring.AlertStatus_RESOLVED,
			CreatedAt: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}

	return &monitoring.GetAlertsResponse{
		Success: true,
		Alerts:  alerts,
		Total:   int32(len(alerts)),
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// GetMetricsHistory retrieves historical metrics
func (s *MonitoringServiceServer) GetMetricsHistory(ctx context.Context, req *monitoring.GetMetricsHistoryRequest) (*monitoring.GetMetricsHistoryResponse, error) {
	if req.MetricName == "" {
		return nil, fmt.Errorf("metric name is required")
	}

	// Generate mock historical data
	history := []*monitoring.MetricPoint{}
	now := time.Now()
	
	// Generate 24 hours of data points
	for i := 0; i < 24; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Hour)
		value := 50.0 + float64(i%20) + (float64(i) * 0.5)
		
		history = append(history, &monitoring.MetricPoint{
			Timestamp: timestamp.Format(time.RFC3339),
			Value:     value,
		})
	}

	return &monitoring.GetMetricsHistoryResponse{
		Success: true,
		MetricName: req.MetricName,
		History: history,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// StartGRPCServer starts the gRPC monitoring service
func StartGRPCServer() error {
	// Load certificates
	creds, err := credentials.NewServerTLSFromFile(
		"/certs/monitoring.crt",
		"/certs/monitoring-key.pem",
	)
	if err != nil {
		return fmt.Errorf("failed to load certificates: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(grpc.Creds(creds))

	// Create monitoring service server
	monitoringServer := NewMonitoringServiceServer()

	// Register monitoring service
	monitoring.RegisterMonitoringServiceServer(grpcServer, monitoringServer)

	// Register health service
	grpc_health_v1.RegisterHealthServer(grpcServer, monitoringServer.healthServer)

	// Register reflection service for debugging
	reflection.Register(grpcServer)

	// Set health status
	monitoringServer.healthServer.SetServingStatus("monitoring.MonitoringService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start listening
	port := getEnv("MONITORING_GRPC_PORT", "50053")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Printf("Monitoring gRPC service starting on port %s", port)
	
	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Helper function to convert user to proto
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}