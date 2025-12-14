//go:build ignore

package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    password := "password"
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Hash: %s\n", hash)
    
    // Verify against existing hash
    existing := "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi"
    err = bcrypt.CompareHashAndPassword([]byte(existing), []byte(password))
    if err != nil {
        fmt.Printf("Existing hash mismatch: %v\n", err)
    } else {
        fmt.Println("Existing hash matches password123")
    }
    
    // Verify new hash
    err = bcrypt.CompareHashAndPassword(hash, []byte(password))
    if err != nil {
        fmt.Printf("New hash mismatch: %v\n", err)
    } else {
        fmt.Println("New hash matches")
    }
}