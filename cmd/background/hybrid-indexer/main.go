package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"problem-search/internal/background/indexing"
	"problem-search/internal/config"
	"problem-search/internal/embedding"
	"problem-search/internal/qdrant"
	"problem-search/internal/storage"
	"problem-search/internal/text"
)

const (
	collectionName     = "leetcode-problems-hybrid-v1"
	embeddingDimension = 1536
	embeddingBatchSize = 96
	indexPageSize      = 100
	indexTimeout       = 30 * time.Minute
)

func main() {
	_ = godotenv.Load()

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

	embeddingClient, err := embedding.NewCohereClient(os.Getenv("COHERE_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	indexCtx, cancel := context.WithTimeout(context.Background(), indexTimeout)
	defer cancel()

	if err := vectorStore.HealthCheck(indexCtx); err != nil {
		log.Fatal(err)
	}
	if err := vectorStore.EnsureHybridCollection(indexCtx, collectionName, embeddingDimension); err != nil {
		log.Fatal(err)
	}

	problems, err := store.ListAllProblems(indexCtx, indexPageSize)
	if err != nil {
		log.Fatal(err)
	}

	indexer, err := indexing.NewBM25Indexer(1.2, 0.75)
	if err != nil {
		log.Fatal(err)
	}

	buildResult, err := indexer.Build(problems)
	if err != nil {
		log.Fatal(err)
	}

	indexPath := filepath.Join("indexes", "bm25-v1.json")
	if err := buildResult.Index.Save(indexPath); err != nil {
		log.Fatal(err)
	}
	log.Printf("saved BM25 index to %s", indexPath)

	for start := 0; start < len(problems); start += embeddingBatchSize {
		end := start + embeddingBatchSize
		if end > len(problems) {
			end = len(problems)
		}

		texts := make([]string, 0, end-start)
		for _, p := range problems[start:end] {
			texts = append(texts, text.SearchableProblem(p))
		}

		embeddings, err := embeddingClient.EmbedTexts(indexCtx, texts, "search_document")
		if err != nil {
			log.Fatalf("embed problems %d-%d: %v", start, end, err)
		}

		points := make([]qdrant.HybridPoint, 0, end-start)
		for index, p := range problems[start:end] {
			if p.ID <= 0 {
				log.Fatalf("problem has invalid database ID: %d", p.ID)
			}

			points = append(points, qdrant.HybridPoint{
				ID:         uint64(p.ID),
				BM25Vector: buildResult.ProblemVectors[start+index].Vector,
				Embedding:  embeddings[index],
			})
		}

		if err := vectorStore.UpsertHybridPoints(indexCtx, collectionName, points); err != nil {
			log.Fatalf("upload hybrid points %d-%d: %v", start, end, err)
		}
		log.Printf("indexed hybrid vectors %d/%d", end, len(problems))
	}
}
