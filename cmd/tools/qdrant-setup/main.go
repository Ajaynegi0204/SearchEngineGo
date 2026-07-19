package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"problem-search/internal/qdrant"
)

const collectionName = "leetcode-problems"

func main() {
	_ = godotenv.Load()

	client, err := qdrant.NewClient(
		os.Getenv("QDRANT_URL"),
		os.Getenv("QDRANT_API_KEY"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		log.Fatal(err)
	}
	if err := client.CreateCollection(ctx, collectionName); err != nil {
		log.Fatal(err)
	}

	log.Printf("created Qdrant collection: %s", collectionName)
}
