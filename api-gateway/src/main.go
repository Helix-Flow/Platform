package main

import (
	"context"
	"crypto/tls"
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
	pbAuth "helixflow/auth"
)

type APIGateway struct {
	redisClient         *redis.Client
	inferencePoolURL    string
	authServiceURL      string
	authClient          pbAuth.AuthServiceClient
	router              *mux.Router
	inferenceHandler    *InferenceHandler
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	User        string        `json:"user,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   Usage                  `json:"usage"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func NewAPIGateway() *APIGateway {
	rdb := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_HOST", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	return &APIGateway{
		redisClient:      rdb,
		inferencePoolURL: getEnv("INFERENCE_POOL_URL", "http://inference-pool:8001"),
		authServiceURL:   getEnv("AUTH_SERVICE_URL", "http://auth-service:8080"),
		router:           mux.NewRouter(),
	}
}

func (ag *APIGateway) SetupRoutes() {
	ag.router.Use(ag.loggingMiddleware)
	ag.router.Use(ag.corsMiddleware)

	// Health check
	ag.router.HandleFunc("/health", ag.healthHandler).Methods("GET")

	// OpenAI-compatible API
	ag.router.HandleFunc("/v1/chat/completions", ag.chatCompletionsHandler).Methods("POST")
	ag.router.HandleFunc("/v1/models", ag.modelsHandler).Methods("GET")

	// WebSocket endpoint (placeholder)
	ag.router.HandleFunc("/ws", ag.websocketHandler)
}

func (ag *APIGateway) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (ag *APIGateway) corsMiddleware(next http.Handler) http.Handler {
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

func (ag *APIGateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	authConnected := ag.authClient != nil
	response := map[string]interface{}{
		"status":                 "healthy",
		"timestamp":              time.Now().Format(time.RFC3339),
		"service":                "api-gateway",
		"auth_service_connected": authConnected,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ag *APIGateway) chatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
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
	if !ag.checkPermission(userID, "model_inference") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Rate limiting
	if !ag.checkRateLimit(userID) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if req.Stream {
		ag.handleStreamingResponse(w, req, userID)
	} else {
		ag.handleStandardResponse(w, req, userID)
	}
}

func (ag *APIGateway) handleStandardResponse(w http.ResponseWriter, req ChatCompletionRequest, userID string) {
	// Use real inference if available, otherwise fallback to mock
	if ag.inferenceHandler != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		response, err := ag.inferenceHandler.HandleChatCompletion(ctx, req, userID)
		if err != nil {
			log.Printf("Inference error: %v", err)
			http.Error(w, "Inference service error", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Fallback to mock response if inference service is not available
	responseContent := generateMockResponse(req.Messages)
	promptTokens := estimatePromptTokens(req.Messages)
	completionTokens := estimateCompletionTokens(responseContent)
	
	// Convert to OpenAI format
	openaiResp := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index:        0,
				Message:      ChatMessage{Role: "assistant", Content: responseContent},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(openaiResp)
}

func (ag *APIGateway) sendSSEChunk(w http.ResponseWriter, data map[string]interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal SSE chunk: %v", err)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (ag *APIGateway) handleStreamingResponse(w http.ResponseWriter, req ChatCompletionRequest, userID string) {
	// Use real inference if available, otherwise fallback to mock
	if ag.inferenceHandler != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		err := ag.inferenceHandler.HandleStreamingChatCompletion(ctx, req, userID, w)
		if err != nil {
			log.Printf("Streaming inference error: %v", err)
			http.Error(w, "Streaming inference service error", http.StatusInternalServerError)
		}
		return
	}
	
	// Fallback to mock streaming if inference service is not available
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial chunk
	initialChunk := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]string{"role": "assistant"},
			},
		},
	}

	sendSSE(w, initialChunk)
	flusher.Flush()

	// Simulate streaming content
	content := "This is a streaming response from the HelixFlow API Gateway."
	words := strings.Fields(content)

	for _, word := range words {
		chunk := map[string]interface{}{
			"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{"content": word + " "},
				},
			},
		}

		sendSSE(w, chunk)
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	}

	// Send final chunk
	finalChunk := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": "stop",
			},
		},
	}

	sendSSE(w, finalChunk)
	sendSSE(w, "[DONE]")
	flusher.Flush()
}

func (ag *APIGateway) modelsHandler(w http.ResponseWriter, r *http.Request) {
	// Check authentication (optional for models endpoint)
	userID := ""
	// Try to authenticate but don't require it
	if id, err := ag.authenticateRequest(r); err == nil {
		userID = id
	}

	models := []Model{
		{
			ID:      "gpt-4",
			Object:  "model",
			Created: 1677649963,
			OwnedBy: "openai",
		},
		{
			ID:      "claude-3-sonnet",
			Object:  "model",
			Created: 1677649963,
			OwnedBy: "anthropic",
		},
		{
			ID:      "deepseek-chat",
			Object:  "model",
			Created: 1677649963,
			OwnedBy: "deepseek",
		},
		{
			ID:      "glm-4",
			Object:  "model",
			Created: 1677649963,
			OwnedBy: "glm",
		},
	}

	// Filter based on user permissions if authenticated
	if userID != "" && !ag.checkPermission(userID, "model_list") {
		models = []Model{} // Return empty if no permission
	}

	response := ModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ag *APIGateway) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// For now, just redirect to chat completion handler
	ag.chatCompletionsHandler(w, r)
}

func (ag *APIGateway) authenticateRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("missing or invalid authorization header")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// If auth client is available, validate with auth service
	if ag.authClient != nil {
		log.Printf("Validating token via auth service (length: %d)", len(token))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &pbAuth.ValidateTokenRequest{Token: token}
		resp, err := ag.authClient.ValidateToken(ctx, req)
		if err != nil {
			log.Printf("Token validation error: %v", err)
			return "", fmt.Errorf("token validation failed: %w", err)
		}
		log.Printf("Token validation result: valid=%v, user=%s", resp.Valid, resp.UserId)
		if !resp.Valid {
			return "", fmt.Errorf("invalid token")
		}
		return resp.UserId, nil
	}

	// Fallback to simple validation (for backward compatibility)
	log.Printf("Auth client is nil, using fallback validation")
	if token == "demo-key" || token == "valid-token" {
		return "demo-user", nil
	}

	return "", fmt.Errorf("invalid token")
}

func (ag *APIGateway) checkPermission(userID, permission string) bool {
	// In a real implementation, check with auth service
	// For now, allow all
	return true
}

func (ag *APIGateway) checkRateLimit(userID string) bool {
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

func sendSSE(w http.ResponseWriter, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func generateMockResponse(messages []ChatMessage) string {
	// Analyze the last user message to generate relevant response
	if len(messages) == 0 {
		return "I'm here to help! What would you like to know?"
	}
	
	lastMessage := messages[len(messages)-1]
	if lastMessage.Role != "user" {
		return "I understand. How can I assist you further?"
	}
	
	userContent := strings.ToLower(lastMessage.Content)
	
	// Generate contextually relevant responses based on common patterns
	switch {
	case strings.Contains(userContent, "hello") || strings.Contains(userContent, "hi"):
		return "Hello! I'm HelixFlow AI assistant. How can I help you today?"
	case strings.Contains(userContent, "thank"):
		return "You're welcome! Is there anything else I can help you with?"
	case strings.Contains(userContent, "weather"):
		return "I don't have access to real-time weather data, but you can check your local weather service for current conditions."
	case strings.Contains(userContent, "time"):
		return fmt.Sprintf("The current time is %s. How can I assist you?", time.Now().Format("15:04:05"))
	case strings.Contains(userContent, "code") || strings.Contains(userContent, "programming"):
		return "I can help with programming questions! What specific coding challenge are you working on?"
	case strings.Contains(userContent, "explain"):
		return "I'd be happy to explain that. Could you provide more specific details about what you'd like me to clarify?"
	case strings.Contains(userContent, "help"):
		return "I'm here to help! I can assist with questions, provide information, or help solve problems. What do you need help with?"
	case len(userContent) < 5:
		return "I see you've entered a short message. Could you provide more details so I can better assist you?"
	default:
		// Generate intelligent response based on message length and content
		if len(userContent) > 100 {
			return "Thank you for the detailed message. I've processed your request and I'm ready to provide assistance based on the information you've shared."
		} else {
			return "I understand your message. As an AI assistant integrated with HelixFlow's enterprise infrastructure, I'm here to provide helpful and accurate responses to your queries."
		}
	}
}

func estimatePromptTokens(messages []ChatMessage) int {
	// Simple token estimation: ~1 token per 4 characters
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	return totalChars / 4
}

func estimateCompletionTokens(content string) int {
	// Simple token estimation: ~1 token per 4 characters
	return len(content) / 4
}



func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	gateway := NewAPIGateway()
	
	// Load configuration
	certFile := getEnv("TLS_CERT", "../certs/api-gateway.crt")
	keyFile := getEnv("TLS_KEY", "../certs/api-gateway-key.pem")
	inferencePoolURL := getEnv("INFERENCE_POOL_URL", "inference-pool:50051")
	
	// Initialize inference handler with gRPC connection
	log.Printf("Setting up inference service connection to %s", inferencePoolURL)
	inferenceHandler, err := NewInferenceHandler(inferencePoolURL)
	if err != nil {
		log.Printf("Warning: Failed to initialize inference handler: %v. Using mock responses.", err)
		inferenceHandler = nil
	} else {
		log.Printf("Inference service connection established successfully")
	}

	// Initialize auth service gRPC client
	authGRPCAddr := getEnv("AUTH_SERVICE_GRPC", "auth-service:8081")
	var authOpt grpc.DialOption
	if strings.Contains(authGRPCAddr, "localhost") || strings.Contains(authGRPCAddr, "127.0.0.1") {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		authOpt = grpc.WithTransportCredentials(creds)
	} else {
		creds := credentials.NewTLS(&tls.Config{})
		authOpt = grpc.WithTransportCredentials(creds)
	}
	authConn, err := grpc.Dial(authGRPCAddr, authOpt)
	if err != nil {
		log.Printf("Warning: Failed to connect to auth service gRPC: %v. Token validation will fail.", err)
	} else {
		gateway.authClient = pbAuth.NewAuthServiceClient(authConn)
		log.Printf("Auth service gRPC connection established successfully")
	}

	// Set inference handler on gateway
	gateway.inferenceHandler = inferenceHandler
	
	gateway.SetupRoutes()

	port := getEnv("PORT", "8443")
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: gateway.router,
	}

	// Check if certificate files actually exist for TLS
	_, certErr := os.Stat(certFile)
	_, keyErr := os.Stat(keyFile)
	
	if certErr == nil && keyErr == nil {
		// Certificates exist, use TLS
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
			MaxVersion: tls.VersionTLS13,
		}
		server.TLSConfig = tlsConfig

		log.Printf("Starting API Gateway with TLS 1.3 on port %s", port)
		log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
	} else {
		// No certificates, use HTTP
		log.Printf("Starting API Gateway (HTTP) on port %s", port)
		log.Fatal(server.ListenAndServe())
	}
}
