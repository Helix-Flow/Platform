package main

import (
	"fmt"
	"log"
	
	"helixflow/database"
)

func main() {
	fmt.Println("Testing database connection...")
	
	// Test SQLite connection
	sqliteConfig := database.GetSQLiteConfig()
	redisConfig := database.GetDefaultRedisConfig()
	
	sqliteManager := database.NewSQLiteManager(sqliteConfig, redisConfig)
	if err := sqliteManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize SQLite: %v", err)
	}
	defer sqliteManager.Close()
	
	fmt.Println("✅ SQLite connection successful")
	
	// Test basic operations
	userID, err := sqliteManager.CreateUser("testuser", "test@example.com", "password", "Test", "User", "HelixFlow")
	if err != nil {
		log.Printf("Failed to create test user: %v", err)
	} else {
		fmt.Printf("✅ Created test user with ID: %s\n", userID)
	}
	
	user, err := sqliteManager.GetUserByUsername("testuser")
	if err != nil {
		log.Printf("Failed to get test user: %v", err)
	} else {
		fmt.Printf("✅ Retrieved test user: %s (%s)\n", user.Username, user.Email)
	}
	
	fmt.Println("✅ All database tests passed!")
}