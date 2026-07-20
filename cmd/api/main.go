package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"problem-search/internal/api"
	"problem-search/internal/auth"
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
	searchLimit    = 50
	resultLimit    = 20
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	startupContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := storage.NewPostgresStore(startupContext, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	vectorStore, err := qdrant.NewClient(cfg.QdrantURL, cfg.QdrantAPIKey)
	if err != nil {
		log.Fatal(err)
	}
	defer vectorStore.Close()

	embeddingClient, err := embedding.NewCohereClient(cfg.CohereAPIKey)
	if err != nil {
		log.Fatal(err)
	}

	rerankerClient, err := rerank.NewCohereClient(cfg.CohereAPIKey)
	if err != nil {
		log.Fatal(err)
	}

	bm25IndexPath := filepath.Join("indexes", "bm25-v1.json")
	bm25Index, err := retrieval.LoadBM25Index(bm25IndexPath)
	if err != nil {
		log.Fatal(err)
	}

	searchEngine, err := search.NewHybridEngine(
		store,
		vectorStore,
		embeddingClient,
		rerankerClient,
		bm25Index,
		collectionName,
		searchLimit,
		resultLimit,
	)
	if err != nil {
		log.Fatal(err)
	}

	authService, err := auth.NewService(store, cfg.JWTSecret)
	if err != nil {
		log.Fatal(err)
	}

	port := cfg.Port

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           api.NewRouter(authService, searchEngine),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("API server listening on http://localhost:%d", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
