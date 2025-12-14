package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"helixflow/inference"
)

// InferenceHandler handles inference requests to the inference pool
type InferenceHandler struct {
	inferenceClient inference.InferenceServiceClient
	timeout         time.Duration
}

// NewInferenceHandler creates a new inference handler
func NewInferenceHandler(inferencePoolURL string) (*InferenceHandler, error) {
	// Create gRPC connection to inference pool
	creds, err := credentials.NewClientTLSFromFile("/certs/inference-pool.crt", "inference-pool")
	if err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
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
		ModelId:     req.Model,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: float32(req.Temperature),
		TopP:        0.9,
		TopK:        50,
		Stream:      req.Stream,
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
		ModelId:     req.Model,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: float32(req.Temperature),
		TopP:        0.9,
		TopK:        50,
		Stream:      true,
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
	encoder := json.NewEncoder(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer doesn't support flushing")
	}

	responseID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()

	for {
		response, err := stream.Recv()
		if err != nil {
			break
		}

		// Convert to SSE format
		sseEvent := map[string]interface{}{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"content": response.Content,
					},
					"finish_reason": nil,
				},
			},
		}

		// Write SSE data
		fmt.Fprintf(w, "data: %s\n\n", mustMarshalJSON(sseEvent))
		flusher.Flush()

		// Check if this is the final chunk
		if response.IsFinal {
			break
		}
	}

	// Send final chunk with finish reason
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