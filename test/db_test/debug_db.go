package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("Testing database connection...")
	
	// Check current directory
	pwd, _ := os.Getwd()
	fmt.Printf("Current directory: %s\n", pwd)
	
	// Check if data directory exists
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		fmt.Println("data directory does not exist")
	} else {
		fmt.Println("data directory exists")
	}
	
	// Check if database file exists
	if _, err := os.Stat("data/helixflow.db"); os.IsNotExist(err) {
		fmt.Println("database file does not exist")
	} else {
		fmt.Println("database file exists")
	}
	
	// Try to open database
	db, err := sql.Open("sqlite3", "data/helixflow.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	
	fmt.Printf("Database opened: %v\n", db != nil)
	
	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	
	fmt.Println("✅ Database connection successful")
}