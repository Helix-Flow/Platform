package main_test

import (
	"testing"
	"github.com/google/uuid"
)

func TestUUIDGeneration(t *testing.T) {
	id := uuid.New()
	// Basic validation that it's not empty
	if id.String() == "" {
		t.Fatal("Generated UUID is empty")
	}
}

func TestUUIDVersion4(t *testing.T) {
	id := uuid.New()
	// Verify it's a UUID v4
	if id.Version() != 4 {
		t.Fatalf("Expected UUID version 4, got version %d", id.Version())
	}
}