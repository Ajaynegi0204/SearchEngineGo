package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"problem-search/internal/api"
	"problem-search/internal/auth"
	"problem-search/internal/config"
	"problem-search/internal/embedding"
	"problem-search/internal/qdrant"
	"problem-search/internal/rerank"
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
	_ = godotenv.Load()

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

	authService, err := auth.NewService(store, os.Getenv("JWT_SECRET"))
	if err != nil {
		log.Fatal(err)
	}

	port := 8080
	if configuredPort := os.Getenv("PORT"); configuredPort != "" {
		port, err = strconv.Atoi(configuredPort)
		if err != nil || port < 1 || port > 65535 {
			log.Fatal("PORT must be a valid TCP port")
		}
	}

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
