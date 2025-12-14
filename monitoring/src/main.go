package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Create monitoring service server
	monitoringServer := NewMonitoringServiceServer()

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register monitoring service
	monitoring.RegisterMonitoringServiceServer(grpcServer, monitoringServer)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Get port from environment
	port := getEnv("MONITORING_PORT", "8083")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Starting Monitoring Service on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
