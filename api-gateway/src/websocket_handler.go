package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"helixflow/inference"
)

// WebSocketManager manages WebSocket connections and real-time communication
type WebSocketManager struct {
	upgrader websocket.Upgrader
	
	// Connection management
	connections map[string]*WebSocketConnection
	mutex       sync.RWMutex
	
	// Message handling
	broadcast chan *WebSocketMessage
	
	// Configuration
	maxConnections int
	pingInterval   time.Duration
	writeTimeout   time.Duration
}

// WebSocketConnection represents a WebSocket client connection
type WebSocketConnection struct {
	ID           string
	Conn         *websocket.Conn
	UserID       string
	ClientIP     string
	UserAgent    string
	ConnectedAt  time.Time
	LastPing     time.Time
	
	// Channels for communication
	send     chan []byte
	close    chan struct{}
	
	// Context for cleanup
	ctx      context.Context
	cancel   context.CancelFunc
}

// WebSocketMessage represents a message structure for WebSocket communication
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
}

// ChatMessage represents a chat message in WebSocket format
type ChatMessageWS struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamingRequest represents a streaming inference request over WebSocket
type StreamingRequest struct {
	Model       string         `json:"model"`
	Messages    []ChatMessageWS `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Stream      bool           `json:"stream"`
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	wsm := &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin in development
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections:    make(map[string]*WebSocketConnection),
		broadcast:      make(chan *WebSocketMessage, 100),
		maxConnections: 1000,
		pingInterval:   30 * time.Second,
		writeTimeout:   10 * time.Second,
	}
	
	// Start broadcast handler
	go wsm.handleBroadcasts()
	
	return wsm
}

// HandleWebSocket handles WebSocket connections
func (wsm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	userID, err := wsm.authenticateWebSocket(r)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}
	
	// Check connection limit
	if wsm.getConnectionCount() >= wsm.maxConnections {
		http.Error(w, "Connection limit exceeded", http.StatusServiceUnavailable)
		return
	}
	
	// Upgrade HTTP connection to WebSocket
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	
	// Create new connection
	ctx, cancel := context.WithCancel(context.Background())
	wsConn := &WebSocketConnection{
		ID:          generateConnectionID(),
		Conn:        conn,
		UserID:      userID,
		ClientIP:    r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
		send:        make(chan []byte, 256),
		close:       make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Register connection
	wsm.registerConnection(wsConn)
	
	// Start connection handlers
	go wsm.handleConnection(wsConn)
	go wsm.handleSend(wsConn)
	go wsm.handlePing(wsConn)
}

// HandleStreamingChat handles streaming chat completions over WebSocket
func (wsm *WebSocketManager) HandleStreamingChat(conn *WebSocketConnection, request StreamingRequest) {
	log.Printf("Starting streaming chat for connection %s, model: %s", conn.ID, request.Model)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(conn.ctx, 5*time.Minute)
	defer cancel()
	
	// Convert WebSocket request to gRPC format
	grpcRequest := &inference.InferenceRequest{
		ModelId:     request.Model,
		MaxTokens:   int32(request.MaxTokens),
		Temperature: float32(request.Temperature),
		TopP:        0.9,
		TopK:        50,
		Stream:      true,
		Messages:    convertWebSocketMessages(request.Messages),
	}
	
	// Send acknowledgment
	ackMessage := WebSocketMessage{
		Type:      "stream_start",
		ID:        generateMessageID(),
		Data:      map[string]interface{}{"model": request.Model},
		Timestamp: time.Now(),
		UserID:    conn.UserID,
	}
	
	conn.sendMessage(ackMessage)
	
	// Simulate streaming inference (in real implementation, connect to inference service)
	wsm.simulateStreamingInference(ctx, conn, grpcRequest)
}

// simulateStreamingInference simulates streaming inference responses
func (wsm *WebSocketManager) simulateStreamingInference(ctx context.Context, conn *WebSocketConnection, request *inference.InferenceRequest) {
	// In a real implementation, this would connect to the inference pool service
	// For now, we'll simulate realistic streaming responses
	
	responseID := generateMessageID()
	created := time.Now().Unix()
	
	// Send initial chunk
	initialMessage := WebSocketMessage{
		Type: "stream_chunk",
		ID:   responseID,
		Data: map[string]interface{}{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   request.ModelId,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{"role": "assistant"},
				},
			},
		},
		Timestamp: time.Now(),
		UserID:    conn.UserID,
	}
	
	conn.sendMessage(initialMessage)
	
	// Simulate streaming content with realistic delays
	responses := wsm.generateStreamingResponses(request)
	
	for i, content := range responses {
		select {
		case <-ctx.Done():
			log.Printf("Streaming context cancelled for connection %s", conn.ID)
			return
		case <-conn.ctx.Done():
			log.Printf("Connection %s closed during streaming", conn.ID)
			return
		default:
			// Send content chunk
			chunkMessage := WebSocketMessage{
				Type: "stream_chunk",
				ID:   responseID,
				Data: map[string]interface{}{
					"id":      responseID,
					"object":  "chat.completion.chunk",
					"created": created,
					"model":   request.ModelId,
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"delta": map[string]interface{}{"content": content},
						},
					},
				},
				Timestamp: time.Now(),
				UserID:    conn.UserID,
			}
			
			conn.sendMessage(chunkMessage)
			
			// Simulate processing delay
			time.Sleep(50 * time.Millisecond)
		}
	}
	
	// Send final chunk
	finalMessage := WebSocketMessage{
		Type: "stream_end",
		ID:   responseID,
		Data: map[string]interface{}{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   request.ModelId,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]interface{}{},
					"finish_reason": "stop",
				},
			},
		},
		Timestamp: time.Now(),
		UserID:    conn.UserID,
	}
	
	conn.sendMessage(finalMessage)
	
	// Send usage statistics
	usageMessage := WebSocketMessage{
		Type: "stream_usage",
		ID:   responseID,
		Data: map[string]interface{}{
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": len(responses) * 5,
				"total_tokens":      10 + len(responses)*5,
			},
		},
		Timestamp: time.Now(),
		UserID:    conn.UserID,
	}
	
	conn.sendMessage(usageMessage)
	
	log.Printf("Streaming chat completed for connection %s", conn.ID)
}

// generateStreamingResponses generates realistic streaming responses based on model
func (wsm *WebSocketManager) generateStreamingResponses(request *inference.InferenceRequest) []string {
	model := request.ModelId
	
	// Get the last user message
	var lastMessage string
	if len(request.Messages) > 0 {
		lastMessage = request.Messages[len(request.Messages)-1].Content
	}
	
	switch model {
	case "gpt-3.5-turbo":
		return wsm.generateGPT35StreamingResponse(lastMessage)
	case "gpt-4":
		return wsm.generateGPT4StreamingResponse(lastMessage)
	case "claude-v1":
		return wsm.generateClaudeStreamingResponse(lastMessage)
	case "llama-2-70b":
		return wsm.generateLlamaStreamingResponse(lastMessage)
	default:
		return wsm.generateDefaultStreamingResponse(lastMessage)
	}
}

// Model-specific streaming response generators
func (wsm *WebSocketManager) generateGPT35StreamingResponse(message string) []string {
	response := "Hello! I'm powered by GPT-3.5 Turbo through HelixFlow's enterprise platform. I'm here to help you with various tasks including answering questions, providing explanations, and assisting with creative writing. How can I assist you today?"
	return strings.Fields(response)
}

func (wsm *WebSocketManager) generateGPT4StreamingResponse(message string) []string {
	response := "Greetings! I'm powered by GPT-4 through HelixFlow's enterprise platform. I offer enhanced reasoning capabilities and can provide more detailed, nuanced responses. How may I help you today?"
	return strings.Fields(response)
}

func (wsm *WebSocketManager) generateClaudeStreamingResponse(message string) []string {
	response := "Hello! I'm Claude, accessible through HelixFlow's unified API. I'm designed to be helpful, harmless, and honest in my interactions. How can I assist you today?"
	return strings.Fields(response)
}

func (wsm *WebSocketManager) generateLlamaStreamingResponse(message string) []string {
	response := "Hello! I'm powered by Llama 2 70B through HelixFlow. I'm an open-source large language model that can help with various tasks including answering questions and providing information."
	return strings.Fields(response)
}

func (wsm *WebSocketManager) generateDefaultStreamingResponse(message string) []string {
	response := "Hello! I understand you're asking about this topic. I'm here to help you with any questions or tasks you have. How can I assist you further?"
	return strings.Fields(response)
}

// Connection Management Functions

func (wsm *WebSocketManager) registerConnection(conn *WebSocketConnection) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	
	wsm.connections[conn.ID] = conn
	log.Printf("Registered WebSocket connection %s for user %s", conn.ID, conn.UserID)
}

func (wsm *WebSocketManager) unregisterConnection(connID string) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	
	if conn, exists := wsm.connections[connID]; exists {
		conn.cancel()
		close(conn.send)
		delete(wsm.connections, connID)
		log.Printf("Unregistered WebSocket connection %s", connID)
	}
}

func (wsm *WebSocketManager) getConnectionCount() int {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	return len(wsm.connections)
}

func (wsm *WebSocketManager) getConnection(connID string) (*WebSocketConnection, bool) {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	
	conn, exists := wsm.connections[connID]
	return conn, exists
}

// Connection Handler Functions

func (wsm *WebSocketManager) handleConnection(conn *WebSocketConnection) {
	defer func() {
		wsm.unregisterConnection(conn.ID)
		conn.Conn.Close()
		log.Printf("WebSocket connection %s closed", conn.ID)
	}()
	
	for {
		select {
		case <-conn.close:
			return
		case <-conn.ctx.Done():
			return
		default:
			var message map[string]interface{}
			err := conn.Conn.ReadJSON(&message)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error for connection %s: %v", conn.ID, err)
				}
				return
			}
			
			wsm.handleMessage(conn, message)
		}
	}
}

func (wsm *WebSocketManager) handleSend(conn *WebSocketConnection) {
	ticker := time.NewTicker(wsm.pingInterval)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-conn.send:
			conn.Conn.SetWriteDeadline(time.Now().Add(wsm.writeTimeout))
			if !ok {
				// Channel closed
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error for connection %s: %v", conn.ID, err)
				return
			}
			
		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(wsm.writeTimeout))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket ping error for connection %s: %v", conn.ID, err)
				return
			}
			conn.LastPing = time.Now()
		}
	}
}

func (wsm *WebSocketManager) handlePing(conn *WebSocketConnection) {
	ticker := time.NewTicker(wsm.pingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if time.Since(conn.LastPing) > wsm.pingInterval*2 {
				log.Printf("Connection %s ping timeout, closing", conn.ID)
				conn.close <- struct{}{}
				return
			}
		case <-conn.ctx.Done():
			return
		}
	}
}

func (wsm *WebSocketManager) handleBroadcasts() {
	for message := range wsm.broadcast {
		wsm.broadcastMessage(message)
	}
}

func (wsm *WebSocketManager) broadcastMessage(message *WebSocketMessage) {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}
	
	for _, conn := range wsm.connections {
		select {
		case conn.send <- data:
		default:
			// Channel full, skip this connection
			log.Printf("WebSocket send channel full for connection %s", conn.ID)
		}
	}
}

func (wsm *WebSocketManager) handleMessage(conn *WebSocketConnection, message map[string]interface{}) {
	messageType, ok := message["type"].(string)
	if !ok {
		log.Printf("Invalid message type from connection %s", conn.ID)
		return
	}
	
	switch messageType {
	case "chat_completion":
		wsm.handleChatCompletion(conn, message)
	case "ping":
		wsm.handlePingMessage(conn, message)
	case "subscribe":
		wsm.handleSubscribe(conn, message)
	default:
		log.Printf("Unknown message type '%s' from connection %s", messageType, conn.ID)
	}
}

func (wsm *WebSocketManager) handleChatCompletion(conn *WebSocketConnection, message map[string]interface{}) {
	data, ok := message["data"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid chat completion data from connection %s", conn.ID)
		return
	}
	
	// Convert to streaming request
	var request StreamingRequest
	requestData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal chat completion data: %v", err)
		return
	}
	
	if err := json.Unmarshal(requestData, &request); err != nil {
		log.Printf("Failed to unmarshal chat completion request: %v", err)
		return
	}
	
	// Handle streaming chat completion
	go wsm.HandleStreamingChat(conn, request)
}

func (wsm *WebSocketManager) handlePingMessage(conn *WebSocketConnection, message map[string]interface{}) {
	pongMessage := WebSocketMessage{
		Type:      "pong",
		ID:        generateMessageID(),
		Data:      map[string]interface{}{"timestamp": time.Now().Unix()},
		Timestamp: time.Now(),
		UserID:    conn.UserID,
	}
	
	conn.sendMessage(pongMessage)
}

func (wsm *WebSocketManager) handleSubscribe(conn *WebSocketConnection, message map[string]interface{}) {
	// Handle subscription to different channels/events
	channels, ok := message["channels"].([]interface{})
	if !ok {
		log.Printf("Invalid subscription channels from connection %s", conn.ID)
		return
	}
	
	subscriptionMessage := WebSocketMessage{
		Type:      "subscription_confirmed",
		ID:        generateMessageID(),
		Data:      map[string]interface{}{"channels": channels},
		Timestamp: time.Now(),
		UserID:    conn.UserID,
	}
	
	conn.sendMessage(subscriptionMessage)
}

// Helper Functions

func (conn *WebSocketConnection) sendMessage(message WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message for connection %s: %v", conn.ID, err)
		return
	}
	
	select {
	case conn.send <- data:
	default:
		log.Printf("Send channel full for connection %s", conn.ID)
	}
}

func (wsm *WebSocketManager) authenticateWebSocket(r *http.Request) (string, error) {
	// Extract authentication token from header or query parameter
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	
	if token == "" {
		return "", fmt.Errorf("missing authentication token")
	}
	
	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	
	// In a real implementation, validate the JWT token
	// For now, return a mock user ID
	return "websocket_user_" + generateConnectionID(), nil
}

func convertWebSocketMessages(messages []ChatMessageWS) []*inference.ChatMessage {
	grpcMessages := make([]*inference.ChatMessage, len(messages))
	for i, msg := range messages {
		grpcMessages[i] = &inference.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return grpcMessages
}

func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// WebSocketManagerStats provides statistics about the WebSocket manager
type WebSocketManagerStats struct {
	ActiveConnections int                    `json:"active_connections"`
	TotalConnections  int64                  `json:"total_connections"`
	MessagesSent      int64                  `json:"messages_sent"`
	MessagesReceived  int64                  `json:"messages_received"`
	ErrorsCount       int64                  `json:"errors_count"`
	Uptime            time.Duration          `json:"uptime"`
	ConnectionStats   map[string]interface{} `json:"connection_stats"`
}

// GetStats returns WebSocket manager statistics
func (wsm *WebSocketManager) GetStats() WebSocketManagerStats {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	
	stats := WebSocketManagerStats{
		ActiveConnections: len(wsm.connections),
		ConnectionStats:   make(map[string]interface{}),
	}
	
	// Add connection-specific stats
	for connID, conn := range wsm.connections {
		stats.ConnectionStats[connID] = map[string]interface{}{
			"user_id":      conn.UserID,
			"connected_at": conn.ConnectedAt,
			"last_ping":    conn.LastPing,
			"client_ip":    conn.ClientIP,
		}
	}
	
	return stats
}