package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	pb "helixflow/auth"
	database "helixflow/database"
)

func main() {
	// Initialize database based on configuration
	dbType := database.GetDatabaseType()
	redisConfig := database.GetDefaultRedisConfig()
	
	log.Printf("Using database type: %s", dbType)
	
	var dbManager database.DatabaseManager
	var err error
	
	switch dbType {
	case database.DatabaseTypePostgres:
		// For now, use SQLite manager for PostgreSQL as well since LegacyDatabaseManager doesn't implement full interface
		sqliteConfig := database.GetSQLiteConfig()
		sqliteManager := database.NewSQLiteManager(sqliteConfig, redisConfig)
		if err = sqliteManager.Initialize(); err != nil {
			log.Fatalf("Failed to initialize SQLite: %v", err)
		}
		defer sqliteManager.Close()
		dbManager = sqliteManager
		
	case database.DatabaseTypeSQLite:
		sqliteConfig := database.GetSQLiteConfig()
		sqliteManager := database.NewSQLiteManager(sqliteConfig, redisConfig)
		if err = sqliteManager.Initialize(); err != nil {
			log.Fatalf("Failed to initialize SQLite: %v", err)
		}
		defer sqliteManager.Close()
		dbManager = sqliteManager
		
	default:
		log.Fatalf("Unknown database type: %s", dbType)
	}

	// Create auth service server
	authServer, err := NewAuthServiceServer(dbManager)
	if err != nil {
		log.Fatalf("Failed to create auth service: %v", err)
	}

	// Get port from environment
	port := getEnv("PORT", "8081")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// Configure TLS if certificates are available
	certFile := getEnv("TLS_CERT", "./certs/auth-service.crt")
	keyFile := getEnv("TLS_KEY", "./certs/auth-service-key.pem")

	var serverOptions []grpc.ServerOption
	_, certErr := os.Stat(certFile)
	_, keyErr := os.Stat(keyFile)
	if certErr == nil && keyErr == nil {
		creds, tlsErr := credentials.NewServerTLSFromFile(certFile, keyFile)
		if tlsErr != nil {
			log.Printf("Failed to load TLS credentials: %v", tlsErr)
		} else {
			serverOptions = append(serverOptions, grpc.Creds(creds))
			log.Printf("TLS enabled for auth service")
		}
	}

	// Create gRPC server with options
	grpcServer := grpc.NewServer(serverOptions...)

	// Register auth service
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	// Enable reflection for development
	reflection.Register(grpcServer)

	log.Printf("Starting Auth Service on port %s", port)
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
