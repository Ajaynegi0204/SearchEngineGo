package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"problem-search/internal/embedding"
)

func main() {
	_ = godotenv.Load()

	client, err := embedding.NewCohereClient(os.Getenv("COHERE_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	embeddings, err := client.EmbedTexts(
		ctx,
		[]string{"A problem about finding the shortest path in a graph"},
		"search_document",
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Cohere embedding connection works; dimension=%d", len(embeddings[0]))
}
