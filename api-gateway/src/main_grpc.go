package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	pbInference "helixflow/inference"
	pbAuth "helixflow/auth"
	pbMonitoring "helixflow/monitoring"
)

type APIGatewayGRPC struct {
	redisClient         *redis.Client
	inferenceClient     pbInference.InferenceServiceClient
	authClient          pbAuth.AuthServiceClient
	monitoringClient    pbMonitoring.MonitoringServiceClient
	router              *mux.Router
	tlsConfig           *tls.Config
}

type ModelResponse struct {
	Data   []ModelInfo `json:"data"`
	Object string      `json:"object"`
}

type ModelInfo struct {
	ID         string            `json:"id"`
	Object     string            `json:"object"`
	Created    int64             `json:"created"`
	OwnedBy    string            `json:"owned_by"`
	Permission []ModelPermission `json:"permission"`
}

type ModelPermission struct {
	ID                 string `json:"id"`
	Object             string `json:"object"`
	Created            int64  `json:"created"`
	AllowCreateEngine  bool   `json:"allow_create_engine"`
	AllowSampling      bool   `json:"allow_sampling"`
	AllowLogprobs      bool   `json:"allow_logprobs"`
	AllowSearchIndices bool   `json:"allow_search_indices"`
	AllowView          bool   `json:"allow_view"`
	AllowFineTuning    bool   `json:"allow_fine_tuning"`
	Organization       string `json:"organization"`
	Group              string `json:"group"`
	IsBlocking         bool   `json:"is_blocking"`
}

// NewAPIGatewayGRPC creates a new API Gateway with gRPC clients
func NewAPIGatewayGRPC() *APIGatewayGRPC {
	// Load TLS certificates
	cert, err := tls.LoadX509KeyPair("./certs/api-gateway-client.crt", "./certs/api-gateway-client-key.pem")
	if err != nil {
		log.Fatalf("Failed to load client certificate: %v", err)
	}

	caCert, err := os.ReadFile("./certs/helixflow-ca.pem")
	if err != nil {
		log.Fatalf("Failed to load CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		log.Fatal("Failed to append CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
	}

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_URL", "localhost:6379"),
		Password: "",
		DB:       0,
	})

	return &APIGatewayGRPC{
		redisClient: rdb,
		tlsConfig:   tlsConfig,
		router:      mux.NewRouter(),
	}
}

// InitializeGRPCClients initializes gRPC connections to backend services
func (ag *APIGatewayGRPC) InitializeGRPCClients() error {
	// Create gRPC credentials
	creds := credentials.NewTLS(ag.tlsConfig)

	// Connect to Inference Pool service
	inferenceConn, err := grpc.Dial(
		getEnv("INFERENCE_POOL_GRPC", "inference-pool:8443"),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to inference pool: %w", err)
	}
	ag.inferenceClient = pbInference.NewInferenceServiceClient(inferenceConn)

	// Connect to Auth Service
	authAddr := getEnv("AUTH_SERVICE_GRPC", "auth-service:8081")
	var authOpt grpc.DialOption
	if strings.Contains(authAddr, "localhost") || strings.Contains(authAddr, "127.0.0.1") {
		authOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		authOpt = grpc.WithTransportCredentials(creds)
	}
	authConn, err := grpc.Dial(authAddr, authOpt)
	if err != nil {
		return fmt.Errorf("failed to connect to auth service: %w", err)
	}
	ag.authClient = pbAuth.NewAuthServiceClient(authConn)

	// Connect to Monitoring Service
	monitoringConn, err := grpc.Dial(
		getEnv("MONITORING_SERVICE_GRPC", "monitoring:8443"),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to monitoring service: %w", err)
	}
	ag.monitoringClient = pbMonitoring.NewMonitoringServiceClient(monitoringConn)

	log.Println("gRPC clients initialized successfully")
	return nil
}

// SetupRoutes sets up HTTP routes
func (ag *APIGatewayGRPC) SetupRoutes() {
	ag.router.Use(ag.loggingMiddleware)
	ag.router.Use(ag.corsMiddleware)

	// Health check
	ag.router.HandleFunc("/health", ag.healthHandler).Methods("GET")

	// OpenAI-compatible endpoints
	ag.router.HandleFunc("/v1/chat/completions", ag.chatCompletionsHandler).Methods("POST")
	ag.router.HandleFunc("/v1/models", ag.modelsHandler).Methods("GET")

	// WebSocket support
	ag.router.HandleFunc("/ws", ag.websocketHandler)

	// Static files
	ag.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
}

// loggingMiddleware logs HTTP requests
func (ag *APIGatewayGRPC) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware handles CORS
func (ag *APIGatewayGRPC) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// healthHandler handles health checks
func (ag *APIGatewayGRPC) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service": "api-gateway-grpc",
		"status":  "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// chatCompletionsHandler handles chat completion requests
func (ag *APIGatewayGRPC) chatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate request
	userID, err := ag.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Model == "" || len(req.Messages) == 0 {
		http.Error(w, "Missing required fields: model, messages", http.StatusBadRequest)
		return
	}

	// Check permissions
	if !ag.checkPermission(userID, "inference") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Check rate limit
	if !ag.checkRateLimit(userID) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Handle streaming vs non-streaming
	if req.Stream {
		ag.handleStreamingResponse(w, req, userID)
	} else {
		ag.handleStandardResponse(w, req, userID)
	}
}

// handleStandardResponse handles non-streaming responses
func (ag *APIGatewayGRPC) handleStandardResponse(w http.ResponseWriter, req ChatCompletionRequest, userID string) {
	// Create gRPC request
	grpcReq := &pbInference.InferenceRequest{
		ModelId:     req.Model,
		UserId:      userID,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: req.Temperature,
		Stream:      false,
	}

	// Convert messages
	for _, msg := range req.Messages {
		grpcReq.Messages = append(grpcReq.Messages, &pbInference.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Call inference service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grpcResp, err := ag.inferenceClient.Inference(ctx, grpcReq)
	if err != nil {
		log.Printf("Inference service error: %v", err)
		http.Error(w, "Inference service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Create HTTP response
	response := ChatCompletionResponse{
		ID:      grpcResp.Id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: grpcResp.Choices[0].Message.Content,
				},
				FinishReason: grpcResp.FinishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     int(grpcResp.Usage.PromptTokens),
			CompletionTokens: int(grpcResp.Usage.CompletionTokens),
			TotalTokens:      int(grpcResp.Usage.TotalTokens),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStreamingResponse handles streaming responses
func (ag *APIGatewayGRPC) handleStreamingResponse(w http.ResponseWriter, req ChatCompletionRequest, userID string) {
	// Create gRPC request
	grpcReq := &pbInference.InferenceRequest{
		ModelId:     req.Model,
		UserId:      userID,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: req.Temperature,
		Stream:      true,
	}

	// Convert messages
	for _, msg := range req.Messages {
		grpcReq.Messages = append(grpcReq.Messages, &pbInference.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Set up streaming response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Call streaming inference service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := ag.inferenceClient.StreamInference(ctx, grpcReq)
	if err != nil {
		log.Printf("Streaming inference service error: %v", err)
		http.Error(w, "Streaming inference service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Process streaming response
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}

		// Create SSE data
		data := map[string]interface{}{
			"id":      chunk.Id,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"content": chunk.Choices[0].Delta.Content,
					},
					"finish_reason": chunk.Choices[0].FinishReason,
				},
			},
		}

		ag.sendSSEChunk(w, data)
		flusher.Flush()
	}

	// Send final chunk
	finalData := map[string]interface{}{
		"id":      req.Model,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index":        0,
				"delta":        map[string]interface{}{},
				"finish_reason": "stop",
			},
		},
	}
	ag.sendSSEChunk(w, finalData)
}

// sendSSEChunk sends an SSE chunk
func (ag *APIGatewayGRPC) sendSSEChunk(w http.ResponseWriter, data map[string]interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

// modelsHandler handles model listing
func (ag *APIGatewayGRPC) modelsHandler(w http.ResponseWriter, r *http.Request) {
	// Call monitoring service to get available models
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pbMonitoring.GetSystemMetricsRequest{}
	_, err := ag.monitoringClient.GetSystemMetrics(ctx, req)
	if err != nil {
		log.Printf("Monitoring service error: %v", err)
		// Return default models on error
		ag.sendDefaultModels(w)
		return
	}

	// Return default models for now
	models := []ModelInfo{
		{
			ID:      "gpt-4",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
		{
			ID:      "gpt-3.5-turbo",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
		{
			ID:      "claude-3-sonnet",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
		{
			ID:      "deepseek-chat",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
	}

	response := ModelResponse{
		Data:   models,
		Object: "list",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendDefaultModels sends default models when monitoring service is unavailable
func (ag *APIGatewayGRPC) sendDefaultModels(w http.ResponseWriter) {
	models := []ModelInfo{
		{
			ID:      "gpt-4",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
		{
			ID:      "gpt-3.5-turbo",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
		{
			ID:      "claude-3-sonnet",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
		{
			ID:      "deepseek-chat",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "helixflow",
		},
	}

	response := ModelResponse{
		Data:   models,
		Object: "list",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// websocketHandler handles WebSocket connections
func (ag *APIGatewayGRPC) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// For now, return a simple response
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("WebSocket support coming soon"))
}

// authenticateRequest authenticates the request
func (ag *APIGatewayGRPC) authenticateRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	// Extract token from "Bearer <token>"
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := tokenParts[1]

	// Validate token with auth service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pbAuth.ValidateTokenRequest{Token: token}
	resp, err := ag.authClient.ValidateToken(ctx, req)
	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}

	if !resp.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return resp.UserId, nil
}

// checkPermission checks if user has required permission
func (ag *APIGatewayGRPC) checkPermission(userID, permission string) bool {
	// Get user permissions from auth service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// For now, use the GetUserPermissions method instead
	permReq := &pbAuth.GetUserPermissionsRequest{UserId: userID}
	permResp, err := ag.authClient.GetUserPermissions(ctx, permReq)
	if err != nil {
		log.Printf("Failed to get user permissions: %v", err)
		return false
	}

	for _, perm := range permResp.Permissions {
		if perm == permission {
			return true
		}
	}

	return false
}

// checkRateLimit checks rate limiting
func (ag *APIGatewayGRPC) checkRateLimit(userID string) bool {
	// Simple rate limiting
	currentMinute := time.Now().Truncate(time.Minute)
	key := fmt.Sprintf("rate_limit:%s:%s", userID, currentMinute.Format(time.RFC3339))

	count, err := ag.redisClient.Incr(context.Background(), key).Result()
	if err != nil {
		return true // Allow on error
	}

	ag.redisClient.Expire(context.Background(), key, time.Minute)

	// Allow 100 requests per minute
	return count <= 100
}

// Start starts the API Gateway
func (ag *APIGatewayGRPC) Start() error {
	// Initialize gRPC clients
	if err := ag.InitializeGRPCClients(); err != nil {
		return fmt.Errorf("failed to initialize gRPC clients: %w", err)
	}

	// Setup routes
	ag.SetupRoutes()

	// Load TLS certificates for HTTPS server
	cert, err := tls.LoadX509KeyPair("/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/certs/api-gateway.crt", "/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/certs/api-gateway-key.pem")
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}

	server := &http.Server{
		Addr:         ":8443",
		Handler:      ag.router,
		TLSConfig:    &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		},
	}

	log.Println("API Gateway gRPC version starting on :8443")
	return server.ListenAndServeTLS("", "")
}

// Main function for the gRPC version
func mainGRPC() {
	gateway := NewAPIGatewayGRPC()
	if err := gateway.Start(); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}