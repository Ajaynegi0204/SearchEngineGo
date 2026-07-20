package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"problem-search/internal/clients/embedding"
	"problem-search/internal/clients/qdrant"
	"problem-search/internal/clients/rerank"
	"problem-search/internal/config"
	"problem-search/internal/retrieval"
	"problem-search/internal/search"
	"problem-search/internal/storage"
)

const (
	collectionName = "leetcode-problems-hybrid-v1"
	candidateLimit = 50
	resultLimit    = 20
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/hybrid-search \"your search text\"")
	}

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, err := storage.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	vectorStore, err := qdrant.NewClient(os.Getenv("QDRANT_URL"), os.Getenv("QDRANT_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	defer vectorStore.Close()

	embeddingClient, err := embedding.NewCohereClient(os.Getenv("COHERE_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	rerankerClient, err := rerank.NewCohereClient(os.Getenv("COHERE_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	bm25Index, err := retrieval.LoadBM25Index("indexes/bm25-v1.json")
	if err != nil {
		log.Fatal(err)
	}

	engine, err := search.NewHybridEngine(
		store,
		vectorStore,
		embeddingClient,
		rerankerClient,
		bm25Index,
		collectionName,
		candidateLimit,
		resultLimit,
	)
	if err != nil {
		log.Fatal(err)
	}

	query := strings.Join(os.Args[1:], " ")
	results, err := engine.Search(ctx, query)
	if err != nil {
		log.Fatal(err)
	}

	for rank, result := range results {
		fmt.Printf(
			"%d. %s | slug=%s | rerank=%.5f | rrf=%.5f | bm25_rank=%d | embedding_rank=%d\n",
			rank+1,
			result.Problem.Title,
			result.Problem.Slug,
			result.RerankScore,
			result.RRFScore,
			result.BM25Rank,
			result.EmbeddingRank,
		)
	}
}
