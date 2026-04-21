package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"github.com/gorilla/websocket"
)

// WebSocketStreamingChallenge validates WebSocket streaming at the API Gateway.
type WebSocketStreamingChallenge struct {
	challenge.BaseChallenge
	cfg *HelixFlowConfig
}

func NewWebSocketStreamingChallenge(cfg *HelixFlowConfig) *WebSocketStreamingChallenge {
	return &WebSocketStreamingChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"gateway-websocket",
			"WebSocket Streaming",
			"Validates WebSocket endpoint accepts connections and streams data for chat completions",
			"gateway",
			[]challenge.ID{"gateway-chat"},
		),
		cfg: cfg,
	}
}

func (c *WebSocketStreamingChallenge) Configure(config *challenge.Config) error {
	return c.BaseChallenge.Configure(config)
}
func (c *WebSocketStreamingChallenge) Validate(ctx context.Context) error { return nil }

func (c *WebSocketStreamingChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}
	outputs := map[string]string{}

	// Build WebSocket URL from gateway URL
	wsURL := c.cfg.APIGatewayURL + "/ws"
	if len(wsURL) > 5 && wsURL[:5] == "https" {
		wsURL = "wss" + wsURL[5:]
	} else if len(wsURL) > 4 && wsURL[:4] == "http" {
		wsURL = "ws" + wsURL[4:]
	}

	// Login via auth service for token
	authClient := NewAPIClient(c.cfg.AuthServiceURL, c.cfg)
	_, _, _ = authClient.Login(ctx, c.cfg.Username, c.cfg.Password)

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		TLSClientConfig:  tlsConfig(c.cfg),
	}

	headers := http.Header{}
	if authClient.token != "" {
		headers.Set("Authorization", "Bearer "+authClient.token)
	}

	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		assertions = append(assertions, Assert("websocket_connect", "connected", false,
			fmt.Sprintf("err=%v status=%d", err, 0),
			"WebSocket connection failed"))
		if resp != nil {
			assertions = append(assertions, AssertStatus("websocket_http_status", resp.StatusCode, 101))
		}
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, outputs, fmt.Sprintf("websocket dial failed: %v", err)), nil
	}
	defer conn.Close()

	assertions = append(assertions, Assert("websocket_connect", "connected", true,
		"connected", "WebSocket connection established"))
	if resp != nil {
		assertions = append(assertions, AssertStatus("websocket_http_status", resp.StatusCode, 101))
	}

	// Send a message in the gateway's expected WebSocket format
	msg := map[string]interface{}{
		"type": "chat_completion",
		"data": map[string]interface{}{
			"model":    "gpt-3.5-turbo",
			"messages": []map[string]string{{"role": "user", "content": "Hello"}},
			"stream":   true,
		},
	}
	if err := conn.WriteJSON(msg); err != nil {
		assertions = append(assertions, Assert("websocket_write", "success", false,
			fmt.Sprintf("%v", err), "WebSocket write failed"))
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, outputs, fmt.Sprintf("websocket write failed: %v", err)), nil
	}
	assertions = append(assertions, Assert("websocket_write", "success", true, "ok", "WebSocket write succeeded"))

	// Read at least one message with timeout
	msgReceived := false
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		_, _, err := conn.ReadMessage()
		if err == nil {
			msgReceived = true
		}
		done <- true
	}()

	select {
	case <-done:
	case <-readCtx.Done():
	}

	assertions = append(assertions, Assert("websocket_read", "message received", msgReceived,
		challenge.Ternary(msgReceived, "received", "timeout"),
		challenge.Ternary(msgReceived, "WebSocket received streamed message", "WebSocket read timed out")))

	outputs["websocket_url"] = wsURL
	status := challenge.StatusPassed
	for _, a := range assertions {
		if !a.Passed {
			status = challenge.StatusFailed
		}
	}
	metrics := map[string]challenge.MetricValue{
		"websocket_latency": {Name: "websocket_latency", Value: float64(time.Since(start).Milliseconds()), Unit: "ms"},
	}
	return c.CreateResult(status, start, assertions, metrics, outputs, ""), nil
}

func (c *WebSocketStreamingChallenge) Cleanup(ctx context.Context) error { return nil }
