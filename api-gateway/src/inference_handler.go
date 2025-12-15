package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	inference "helixflow/api-gateway/inference"
)

// InferenceHandler handles inference requests to the inference pool
type InferenceHandler struct {
	inferenceClient inference.InferenceServiceClient
	timeout         time.Duration
}

// NewInferenceHandler creates a new inference handler
func NewInferenceHandler(inferencePoolURL string) (*InferenceHandler, error) {
	var creds credentials.TransportCredentials
	if strings.Contains(inferencePoolURL, "localhost") || strings.Contains(inferencePoolURL, "127.0.0.1") {
		creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	} else {
		// Try to load TLS certificates for secure connections
		certPath := "./certs/inference-pool.crt"
		loadedCreds, err := credentials.NewClientTLSFromFile(certPath, "inference-pool")
		if err != nil {
			return nil, fmt.Errorf("failed to load certificates: %w", err)
		}
		creds = loadedCreds
	}

	conn, err := grpc.Dial(inferencePoolURL, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to inference pool: %w", err)
	}

	client := inference.NewInferenceServiceClient(conn)

	return &InferenceHandler{
		inferenceClient: client,
		timeout:         30 * time.Second,
	}, nil
}

// HandleChatCompletion handles chat completion requests
func (h *InferenceHandler) HandleChatCompletion(ctx context.Context, req ChatCompletionRequest, userID string) (*ChatCompletionResponse, error) {
	// Convert request to inference service format
	inferenceReq := &inference.InferenceRequest{
		Model:       req.Model,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: req.Temperature,
		TopP:        0.9,
		Stream:      req.Stream,
		UserId:      userID,
		Messages:    convertMessages(req.Messages),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Call inference service
	response, err := h.inferenceClient.GenerateCompletion(ctx, inferenceReq)
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Convert response to OpenAI format
	return h.convertToOpenAIFormat(response, req), nil
}

// HandleStreamingChatCompletion handles streaming chat completion requests
func (h *InferenceHandler) HandleStreamingChatCompletion(ctx context.Context, req ChatCompletionRequest, userID string, w http.ResponseWriter) error {
	// Set up streaming response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Convert request to inference service format
	inferenceReq := &inference.InferenceRequest{
		Model:       req.Model,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: req.Temperature,
		TopP:        0.9,
		Stream:      true,
		UserId:      userID,
		Messages:    convertMessages(req.Messages),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Call streaming inference service
	stream, err := h.inferenceClient.GenerateStreamingCompletion(ctx, inferenceReq)
	if err != nil {
		return fmt.Errorf("streaming inference failed: %w", err)
	}

	// Process streaming response
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer doesn't support flushing")
	}

	responseID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()
	sentFinishReason := false

	for {
		response, err := stream.Recv()
		if err != nil {
			break
		}

		// Skip if no choices
		if len(response.Choices) == 0 {
			continue
		}

		// Convert to SSE format
		delta := map[string]interface{}{}
		if response.Choices[0].Delta != nil {
			delta["content"] = response.Choices[0].Delta.Content
		}
		var finishReason interface{} = response.Choices[0].FinishReason
		if response.Choices[0].FinishReason == "" {
			finishReason = nil
		} else {
			sentFinishReason = true
		}
		sseEvent := map[string]interface{}{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         delta,
					"finish_reason": finishReason,
				},
			},
		}

		// Write SSE data
		fmt.Fprintf(w, "data: %s\n\n", mustMarshalJSON(sseEvent))
		flusher.Flush()

		// Check if this is the final chunk
		if response.Choices[0].FinishReason != "" {
			break
		}
	}

	// Send final chunk with finish reason if not already sent
	if !sentFinishReason {
		finalEvent := map[string]interface{}{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index":        0,
					"delta":        map[string]interface{}{},
					"finish_reason": "stop",
				},
			},
		}

		fmt.Fprintf(w, "data: %s\n\n", mustMarshalJSON(finalEvent))
		flusher.Flush()
	}

	return nil
}

// convertMessages converts OpenAI format messages to inference format
func convertMessages(messages []ChatMessage) []*inference.ChatMessage {
	inferenceMessages := make([]*inference.ChatMessage, len(messages))
	for i, msg := range messages {
		inferenceMessages[i] = &inference.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return inferenceMessages
}

// convertToOpenAIFormat converts inference response to OpenAI format
func (h *InferenceHandler) convertToOpenAIFormat(resp *inference.InferenceResponse, req ChatCompletionRequest) *ChatCompletionResponse {
	return &ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index:        0,
				Message:      ChatMessage{Role: "assistant", Content: resp.Choices[0].Message.Content},
				FinishReason: resp.Choices[0].FinishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
	}
}

// mustMarshalJSON marshals JSON with error handling
func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}
	return string(data)
}