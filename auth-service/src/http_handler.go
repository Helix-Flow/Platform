package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	pb "helixflow/auth"
)

// AuthHTTPServer wraps the gRPC auth server to provide HTTP REST API
type AuthHTTPServer struct {
	grpcServer     *AuthServiceServer
	failedLogins   map[string][]time.Time
	failedLoginMu  sync.Mutex
}

// NewAuthHTTPServer creates a new HTTP server for auth
func NewAuthHTTPServer(grpcServer *AuthServiceServer) *AuthHTTPServer {
	return &AuthHTTPServer{
		grpcServer:   grpcServer,
		failedLogins: make(map[string][]time.Time),
	}
}

func (s *AuthHTTPServer) recordFailedLogin(email string) {
	s.failedLoginMu.Lock()
	defer s.failedLoginMu.Unlock()
	
	now := time.Now()
	s.failedLogins[email] = append(s.failedLogins[email], now)
	
	// Clean old entries (> 5 minutes)
	cutoff := now.Add(-5 * time.Minute)
	var recent []time.Time
	for _, t := range s.failedLogins[email] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	s.failedLogins[email] = recent
	
	// Report brute force if > 10 failed attempts in 5 minutes
	if len(recent) > 10 {
		reportIncident(Incident{
			Type:      "brute_force",
			Source:    "auth-service",
			Message:   "Multiple failed login attempts detected for " + email,
			Timestamp: now,
			Severity:  "high",
		})
	}
}

// Start starts the HTTP server on the given port
func (s *AuthHTTPServer) Start(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.loginHandler)
	mux.HandleFunc("/refresh", s.refreshHandler)
	mux.HandleFunc("/revoke", s.revokeHandler)
	mux.HandleFunc("/register", s.registerHandler)
	mux.HandleFunc("/health", s.healthHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Starting Auth HTTP Server on port %s", port)
	return server.ListenAndServe()
}

type Incident struct {
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
}

func reportIncident(incident Incident) {
	monitoringURL := os.Getenv("MONITORING_SERVICE_URL")
	if monitoringURL == "" {
		monitoringURL = "http://localhost:8083"
	}
	data, _ := json.Marshal(incident)
	resp, err := http.Post(monitoringURL+"/api/security/incidents", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to report incident: %v", err)
		return
	}
	defer resp.Body.Close()
}

// loginHandler handles POST /login
func (s *AuthHTTPServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Call gRPC Login method
	ctx := context.Background()
	grpcReq := &pb.LoginRequest{
		Username: req.Email, // email field used as username
		Password: req.Password,
	}
	grpcResp, err := s.grpcServer.Login(ctx, grpcReq)
	if err != nil {
		s.recordFailedLogin(req.Email)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Map response to expected JSON
	response := map[string]interface{}{
		"access_token":  grpcResp.AccessToken,
		"refresh_token": grpcResp.RefreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// refreshHandler handles POST /refresh
func (s *AuthHTTPServer) refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	grpcReq := &pb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	}
	grpcResp, err := s.grpcServer.RefreshToken(ctx, grpcReq)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"access_token":  grpcResp.AccessToken,
		"refresh_token": grpcResp.RefreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// revokeHandler handles POST /revoke
func (s *AuthHTTPServer) revokeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	// Use Logout gRPC method (simplified: token may be access token)
	grpcReq := &pb.LogoutRequest{
		Token: req.Token,
	}
	grpcResp, err := s.grpcServer.Logout(ctx, grpcReq)
	if err != nil {
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}

	if grpcResp.Success {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	} else {
		http.Error(w, "Revocation failed", http.StatusInternalServerError)
	}
}

// registerHandler handles POST /register
func (s *AuthHTTPServer) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username     string `json:"username"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Organization string `json:"organization"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	grpcReq := &pb.RegisterRequest{
		Username:     req.Username,
		Email:        req.Email,
		Password:     req.Password,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Organization: req.Organization,
	}
	grpcResp, err := s.grpcServer.Register(ctx, grpcReq)
	if err != nil {
		log.Printf("Registration failed: %v", err)
		http.Error(w, "Registration failed", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": grpcResp.Success,
		"message": grpcResp.Message,
		"user_id": grpcResp.UserId,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// healthHandler handles GET /health
func (s *AuthHTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "auth-http"})
}