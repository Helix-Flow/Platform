package main

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("Testing SQLite database connection...")
	
	// Open the database file
	db, err := sql.Open("sqlite3", "../../data/helixflow.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	
	fmt.Println("✅ Database connection successful")
	
	// Check if users table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		fmt.Printf("⚠️  Could not query users table: %v\n", err)
	} else {
		fmt.Printf("✅ Users table has %d records\n", count)
	}
	
	fmt.Println("✅ Basic database test completed successfully!")
}