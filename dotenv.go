package main

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadDotEnv() {
	if err := godotenv.Load("main.env"); err != nil {
		log.Printf("⚠️  .env not loaded: %v (run from project root where .env is)", err)
	} else {
		log.Printf("✅ .env loaded")
	}
}
