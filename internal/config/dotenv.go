package config

import (
	"log"

	"github.com/joho/godotenv"
)

// LoadDotEnv tries to load environment variables from file "main.env".
// This is optional: if the file does not exist, we just continue.
//
// Note: in production you usually set env vars in the system, not in a file.
func LoadDotEnv() {
	if err := godotenv.Load("main.env"); err != nil {
		log.Printf("⚠️  main.env not loaded: %v (run from project root)", err)
		return
	}
	log.Printf("✅ main.env loaded")
}
