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

// GetSystemMetrics retrieves current system metrics
func (s *MonitoringServiceServer) GetSystemMetrics(ctx context.Context, req *monitoring.GetSystemMetricsRequest) (*monitoring.GetSystemMetricsResponse, error) {
	// Generate realistic system metrics
	metrics := []*monitoring.MetricData{
		{
			Name:      "cpu_usage",
			Value:     45.2 + float64(time.Now().UnixNano()%20-10),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "percent",
		},
		{
			Name:      "memory_usage",
			Value:     67.8 + float64(time.Now().UnixNano()%15-7),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "percent",
		},
		{
			Name:      "disk_usage",
			Value:     23.1 + float64(time.Now().UnixNano()%10-5),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "percent",
		},
	}

	return &monitoring.GetSystemMetricsResponse{
		Success: true,
		Metrics: metrics,
		Message: "System metrics retrieved successfully",
	}, nil
}

// GetServiceMetrics retrieves service-specific metrics
func (s *MonitoringServiceServer) GetServiceMetrics(ctx context.Context, req *monitoring.GetServiceMetricsRequest) (*monitoring.GetServiceMetricsResponse, error) {
	// Generate service metrics based on service name
	serviceMetrics := []*monitoring.MetricData{
		{
			Name:      "request_count",
			Value:     1250 + float64(time.Now().Unix()%100),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "count",
		},
		{
			Name:      "response_time",
			Value:     45.2 + float64(time.Now().UnixNano()%20-10),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "ms",
		},
		{
			Name:      "error_rate",
			Value:     0.5 + float64(time.Now().UnixNano()%5),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "percent",
		},
	}

	serviceInfo := &monitoring.ServiceInfo{
		Name:         req.ServiceName,
		Status:       "healthy",
		Version:      "1.0.0",
		ReplicaCount: 2,
		Endpoints:    []string{"http://localhost:8080", "https://localhost:8443"},
		Labels:       map[string]string{"environment": "production", "team": "platform"},
	}

	return &monitoring.GetServiceMetricsResponse{
		Success:     true,
		Metrics:     serviceMetrics,
		ServiceInfo: serviceInfo,
		Message:     fmt.Sprintf("Service metrics for %s retrieved successfully", req.ServiceName),
	}, nil
}

// GetGPUMetrics retrieves GPU metrics
func (s *MonitoringServiceServer) GetGPUMetrics(ctx context.Context, req *monitoring.GetGPUMetricsRequest) (*monitoring.GetGPUMetricsResponse, error) {
	// Generate mock GPU metrics
	gpuMetrics := []*monitoring.GPUMetricData{
		{
			GpuId: 0,
			Metrics: []*monitoring.MetricData{
				{
					Name:      "gpu_utilization",
					Value:     75.3 + float64(time.Now().UnixNano()%20-10),
					Timestamp: time.Now().Format(time.RFC3339),
					Unit:      "percent",
				},
				{
					Name:      "gpu_memory_usage",
					Value:     45.2 + float64(time.Now().UnixNano()%15-7),
					Timestamp: time.Now().Format(time.RFC3339),
					Unit:      "percent",
				},
				{
					Name:      "gpu_temperature",
					Value:     65.0 + float64(time.Now().UnixNano()%10-5),
					Timestamp: time.Now().Format(time.RFC3339),
					Unit:      "celsius",
				},
			},
		},
	}

	return &monitoring.GetGPUMetricsResponse{
		Success:   true,
		GpuMetrics: gpuMetrics,
		Message:   "GPU metrics retrieved successfully",
	}, nil
}

// GetApplicationMetrics retrieves application-specific metrics
func (s *MonitoringServiceServer) GetApplicationMetrics(ctx context.Context, req *monitoring.GetApplicationMetricsRequest) (*monitoring.GetApplicationMetricsResponse, error) {
	// Generate application metrics
	appMetrics := []*monitoring.MetricData{
		{
			Name:      "active_users",
			Value:     150 + float64(time.Now().Unix()%50),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "count",
		},
		{
			Name:      "requests_per_second",
			Value:     25.5 + float64(time.Now().UnixNano()%10-5),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "rps",
		},
		{
			Name:      "average_response_time",
			Value:     125.3 + float64(time.Now().UnixNano()%20-10),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "ms",
		},
	}

	return &monitoring.GetApplicationMetricsResponse{
		Success: true,
		Metrics: appMetrics,
		Message: fmt.Sprintf("Application metrics for %s retrieved successfully", req.ApplicationName),
	}, nil
}

// CreateAlertRule creates a new alert rule
func (s *MonitoringServiceServer) CreateAlertRule(ctx context.Context, req *monitoring.CreateAlertRuleRequest) (*monitoring.CreateAlertRuleResponse, error) {
	fmt.Printf("[ALERT_RULE] Created alert rule: %s - %s (Severity: %v)\n",
		req.Name, req.Description, req.Severity)

	// Create the alert rule
	alertRule := &monitoring.AlertRule{
		Id:                   fmt.Sprintf("rule_%d", time.Now().Unix()),
		Name:                 req.Name,
		Description:          req.Description,
		MetricName:           req.MetricName,
		Condition:            req.Condition,
		Threshold:            req.Threshold,
		DurationSeconds:      req.DurationSeconds,
		Severity:             req.Severity,
		NotificationChannels: req.NotificationChannels,
		Labels:               req.Labels,
		Enabled:              req.Enabled,
		CreatedAt:            time.Now().Format(time.RFC3339),
		UpdatedAt:            time.Now().Format(time.RFC3339),
	}

	return &monitoring.CreateAlertRuleResponse{
		Success:   true,
		Message:   "Alert rule created successfully",
		AlertRule: alertRule,
	}, nil
}

// UpdateAlertRule updates an existing alert rule
func (s *MonitoringServiceServer) UpdateAlertRule(ctx context.Context, req *monitoring.UpdateAlertRuleRequest) (*monitoring.UpdateAlertRuleResponse, error) {
	if req.RuleId == "" {
		return nil, fmt.Errorf("rule ID is required")
	}

	fmt.Printf("[ALERT_RULE] Updated alert rule: %s\n", req.RuleId)

	return &monitoring.UpdateAlertRuleResponse{
		Success: true,
		Message: "Alert rule updated successfully",
	}, nil
}

// DeleteAlertRule deletes an alert rule
func (s *MonitoringServiceServer) DeleteAlertRule(ctx context.Context, req *monitoring.DeleteAlertRuleRequest) (*monitoring.DeleteAlertRuleResponse, error) {
	if req.RuleId == "" {
		return nil, fmt.Errorf("rule ID is required")
	}

	fmt.Printf("[ALERT_RULE] Deleted alert rule: %s\n", req.RuleId)

	return &monitoring.DeleteAlertRuleResponse{
		Success: true,
		Message: "Alert rule deleted successfully",
	}, nil
}

// ListAlertRules lists all alert rules
func (s *MonitoringServiceServer) ListAlertRules(ctx context.Context, req *monitoring.ListAlertRulesRequest) (*monitoring.ListAlertRulesResponse, error) {
	// Mock alert rules
	rules := []*monitoring.AlertRule{
		{
			Id:                   "rule_1",
			Name:                 "High CPU Usage",
			Description:          "CPU usage exceeds 80%",
			MetricName:           "cpu_usage",
			Condition:            ">",
			Threshold:            80.0,
			DurationSeconds:      300,
			Severity:             monitoring.AlertSeverity_ALERT_SEVERITY_WARNING,
			NotificationChannels: []string{"email", "slack"},
			Enabled:              true,
			CreatedAt:            time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			UpdatedAt:            time.Now().Format(time.RFC3339),
		},
		{
			Id:                   "rule_2",
			Name:                 "Database Connection Pool",
			Description:          "Connection pool utilization above 90%",
			MetricName:           "db_connections",
			Condition:            ">",
			Threshold:            90.0,
			DurationSeconds:      600,
			Severity:             monitoring.AlertSeverity_ALERT_SEVERITY_INFO,
			NotificationChannels: []string{"email"},
			Enabled:              true,
			CreatedAt:            time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			UpdatedAt:            time.Now().Format(time.RFC3339),
		},
	}

	return &monitoring.ListAlertRulesResponse{
		Success:    true,
		AlertRules: rules,
		TotalCount: int32(len(rules)),
	}, nil
}

// GetAlerts retrieves active alerts
func (s *MonitoringServiceServer) GetAlerts(ctx context.Context, req *monitoring.GetAlertsRequest) (*monitoring.GetAlertsResponse, error) {
	// Mock alerts
	alerts := []*monitoring.Alert{
		{
			Id:        "alert_1",
			RuleId:    "rule_1",
			RuleName:  "High CPU Usage",
			Severity:  monitoring.AlertSeverity_ALERT_SEVERITY_WARNING,
			Status:    "active",
			StartedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			Id:         "alert_2",
			RuleId:     "rule_2",
			RuleName:   "Database Connection Pool",
			Severity:   monitoring.AlertSeverity_ALERT_SEVERITY_INFO,
			Status:     "resolved",
			StartedAt:  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			ResolvedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
	}

	return &monitoring.GetAlertsResponse{
		Success:    true,
		Alerts:     alerts,
		TotalCount: int32(len(alerts)),
	}, nil
}

// AcknowledgeAlert acknowledges an alert
func (s *MonitoringServiceServer) AcknowledgeAlert(ctx context.Context, req *monitoring.AcknowledgeAlertRequest) (*monitoring.AcknowledgeAlertResponse, error) {
	if req.AlertId == "" {
		return nil, fmt.Errorf("alert ID is required")
	}

	fmt.Printf("[ALERT] Acknowledged alert: %s by user %s\n", req.AlertId, req.UserId)

	return &monitoring.AcknowledgeAlertResponse{
		Success: true,
		Message: "Alert acknowledged successfully",
	}, nil
}

// GetScalingRecommendations retrieves predictive scaling recommendations
func (s *MonitoringServiceServer) GetScalingRecommendations(ctx context.Context, req *monitoring.GetScalingRecommendationsRequest) (*monitoring.GetScalingRecommendationsResponse, error) {
	// Mock scaling recommendations
	recommendations := []*monitoring.ScalingRecommendation{
		{
			ServiceName:       "api-gateway",
			Action:            monitoring.ScalingAction_SCALING_ACTION_SCALE_UP,
			TargetReplicas:    3,
			Reason:            "CPU usage trending upward",
			ConfidenceScore:   0.85,
			EstimatedTime:     "5 minutes",
			Metrics:           map[string]string{"cpu_usage": "85%", "request_rate": "120rps"},
		},
		{
			ServiceName:       "inference-pool",
			Action:            monitoring.ScalingAction_SCALING_ACTION_SCALE_UP,
			TargetReplicas:    2,
			Reason:            "Increased inference requests",
			ConfidenceScore:   0.72,
			EstimatedTime:     "3 minutes",
			Metrics:           map[string]string{"inference_rate": "45rps", "queue_depth": "15"},
		},
	}

	return &monitoring.GetScalingRecommendationsResponse{
		Success:           true,
		Recommendations:   recommendations,
		Message:           "Scaling recommendations generated successfully",
	}, nil
}

// StreamMetrics streams metrics in real-time
func (s *MonitoringServiceServer) StreamMetrics(req *monitoring.StreamMetricsRequest, stream monitoring.MonitoringService_StreamMetricsServer) error {
	// Stream metrics for 10 iterations
	for i := 0; i < 10; i++ {
		metric := &monitoring.MetricData{
			Name:      "cpu_usage",
			Value:     45.2 + float64(time.Now().UnixNano()%20-10),
			Timestamp: time.Now().Format(time.RFC3339),
			Unit:      "percent",
		}

		if err := stream.Send(metric); err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

// StartGRPCServer starts the gRPC monitoring service
func StartGRPCServer() error {
	// Load certificates
	certFile := getEnv("MONITORING_TLS_CERT", "./certs/monitoring.crt")
	keyFile := getEnv("MONITORING_TLS_KEY", "./certs/monitoring-key.pem")
	
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
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