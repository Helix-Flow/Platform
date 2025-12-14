package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	pb "helixflow/auth"
)

// AuthHTTPServer wraps the gRPC auth server to provide HTTP REST API
type AuthHTTPServer struct {
	grpcServer *AuthServiceServer
}

// NewAuthHTTPServer creates a new HTTP server for auth
func NewAuthHTTPServer(grpcServer *AuthServiceServer) *AuthHTTPServer {
	return &AuthHTTPServer{
		grpcServer: grpcServer,
	}
}

// Start starts the HTTP server on the given port
func (s *AuthHTTPServer) Start(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.loginHandler)
	mux.HandleFunc("/refresh", s.refreshHandler)
	mux.HandleFunc("/revoke", s.revokeHandler)
	mux.HandleFunc("/health", s.healthHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Starting Auth HTTP Server on port %s", port)
	return server.ListenAndServe()
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

// healthHandler handles GET /health
func (s *AuthHTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "auth-http"})
}