package main

import (
	"fmt"
	"github.com/google/uuid"
)

func main() {
	fmt.Println("Generated UUID for JTI:", uuid.New().String())
}