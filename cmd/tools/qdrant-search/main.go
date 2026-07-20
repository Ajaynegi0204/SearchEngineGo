package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"problem-search/internal/clients/qdrant"
	"problem-search/internal/config"
	"problem-search/internal/retrieval"
	"problem-search/internal/storage"
)

const collectionName = "leetcode-problems"

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/qdrant-search \"your search text\"")
	}

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	index, err := retrieval.LoadBM25Index("indexes/bm25-v1.json")
	if err != nil {
		log.Fatal(err)
	}

	query := strings.Join(os.Args[1:], " ")
	queryVector, err := index.BuildQueryVector(query)
	if err != nil {
		log.Fatal(err)
	}
	if len(queryVector.Indices) == 0 {
		log.Printf("query contains no terms from the BM25 vocabulary: %q", query)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := storage.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	vectorStore, err := qdrant.NewClient(
		os.Getenv("QDRANT_URL"),
		os.Getenv("QDRANT_API_KEY"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer vectorStore.Close()

	results, err := vectorStore.SearchSparseVectors(ctx, collectionName, queryVector, 10)
	if err != nil {
		log.Fatal(err)
	}

	problemIDs := make([]int64, 0, len(results))
	for _, result := range results {
		problemIDs = append(problemIDs, int64(result.ID))
	}

	problems, err := store.GetProblemsByIDs(ctx, problemIDs)
	if err != nil {
		log.Fatal(err)
	}

	for rank, result := range results {
		p, ok := problems[int64(result.ID)]
		if !ok {
			log.Printf("problem metadata not found for ID %d", result.ID)
			continue
		}

		fmt.Printf("%d. %s | slug=%s | score=%.4f\n", rank+1, p.Title, p.Slug, result.Score)
	}
}
