package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting speech-to-speech agent...")
	
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	log.Println("Initializing conversation manager...")
	manager := NewConversationManager()
	log.Println("Conversation manager initialized, starting main loop...")
	if err := manager.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
