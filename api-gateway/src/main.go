package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

type APIGateway struct {
	redisClient      *redis.Client
	inferencePoolURL string
	authServiceURL   string
	router           *mux.Router
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
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "api-gateway",
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
	// Forward to inference pool
	inferenceReq := map[string]interface{}{
		"model":      req.Model,
		"messages":   req.Messages,
		"max_tokens": req.MaxTokens,
		"stream":     false,
		"user_id":    userID,
	}

	jsonData, _ := json.Marshal(inferenceReq)

	resp, err := http.Post(ag.inferencePoolURL+"/inference", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Inference pool error: %v", err)
		http.Error(w, "Inference service unavailable", http.StatusServiceUnavailable)
		return
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Inference pool returned %d: %s", resp.StatusCode, string(body))
		http.Error(w, "Inference failed", http.StatusInternalServerError)
		return
	}

	// Parse inference response
	var inferenceResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&inferenceResp); err != nil {
		http.Error(w, "Invalid inference response", http.StatusInternalServerError)
		return
	}

	// Convert to OpenAI format
	openaiResp := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index:        0,
				Message:      ChatMessage{Role: "assistant", Content: getStringFromMap(inferenceResp, "result")},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     len(fmt.Sprintf("%v", req.Messages)),
			CompletionTokens: 50,
			TotalTokens:      len(fmt.Sprintf("%v", req.Messages)) + 50,
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
	userID, _ := ag.authenticateRequest(r)

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

	// Validate token with auth service
	authReq := map[string]string{"token": token}
	jsonData, _ := json.Marshal(authReq)

	resp, err := http.Post(ag.authServiceURL+"/validate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid token")
	}

	var authResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&authResp)

	userID, ok := authResp["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid auth response")
	}

	return userID, nil
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	gateway := NewAPIGateway()
	gateway.SetupRoutes()

	port := getEnv("PORT", "8443")
	certFile := getEnv("TLS_CERT", "/certs/api-gateway.crt")
	keyFile := getEnv("TLS_KEY", "/certs/api-gateway.key")

	server := &http.Server{
		Addr:    ":" + port,
		Handler: gateway.router,
	}

	// Configure TLS
	if certFile != "" && keyFile != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
			MaxVersion: tls.VersionTLS13,
		}
		server.TLSConfig = tlsConfig

		log.Printf("Starting API Gateway with TLS 1.3 on port %s", port)
		log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
	} else {
		log.Printf("Starting API Gateway (HTTP) on port %s", port)
		log.Fatal(server.ListenAndServe())
	}
}
