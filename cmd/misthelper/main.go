package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file from current directory if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Verify required config is present before starting
	token := os.Getenv("MIST_API_TOKEN")
	if token == "" {
		log.Fatal("MIST_API_TOKEN is required. Set it in .env or as an environment variable.")
	}

	orgID := os.Getenv("MIST_ORG_ID")
	if orgID == "" {
		log.Fatal("MIST_ORG_ID is required. Set it in .env or as an environment variable.")
	}

	fmt.Println("MistHelper-Go starting...")
	fmt.Printf("Org ID: %s\n", orgID)
}
