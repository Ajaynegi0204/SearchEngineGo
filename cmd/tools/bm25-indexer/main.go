package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"problem-search/internal/background/indexing"
	"problem-search/internal/clients/qdrant"
	"problem-search/internal/config"
	"problem-search/internal/storage"
)

const (
	collectionName  = "leetcode-problems"
	qdrantBatchSize = 100
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := storage.NewPostgresStore(startupCtx, cfg.DatabaseURL)
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

	indexCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	problems, err := store.ListAllProblems(indexCtx, 100)
	if err != nil {
		log.Fatal(err)
	}

	indexer, err := indexing.NewBM25Indexer(1.2, 0.75)
	if err != nil {
		log.Fatal(err)
	}

	problemVectors, err := indexer.Build(problems)
	if err != nil {
		log.Fatal(err)
	}

	indexPath := filepath.Join("indexes", "bm25-v1.json")
	if err := problemVectors.Index.Save(indexPath); err != nil {
		log.Fatal(err)
	}

	log.Printf("saved BM25 index to %s", indexPath)
	log.Printf("generated BM25 vectors for %d problems", len(problemVectors.ProblemVectors))

	for start := 0; start < len(problemVectors.ProblemVectors); start += qdrantBatchSize {
		end := start + qdrantBatchSize
		if end > len(problemVectors.ProblemVectors) {
			end = len(problemVectors.ProblemVectors)
		}

		batch := make([]qdrant.SparsePoint, 0, end-start)
		for _, problemVector := range problemVectors.ProblemVectors[start:end] {
			if problemVector.Problem.ID <= 0 {
				log.Fatalf("problem has invalid database ID: %d", problemVector.Problem.ID)
			}

			batch = append(batch, qdrant.SparsePoint{
				ID:     uint64(problemVector.Problem.ID),
				Vector: problemVector.Vector,
			})
		}

		if err := vectorStore.UpsertSparseVectors(indexCtx, collectionName, batch); err != nil {
			log.Fatalf("upload Qdrant batch %d-%d: %v", start, end, err)
		}
		log.Printf("uploaded Qdrant vectors %d/%d", end, len(problemVectors.ProblemVectors))
	}
}
