package main

import (
	"context"
	"log"
	"time"

	"problem-search/internal/background/ingest"
	"problem-search/internal/background/leetcode"
	"problem-search/internal/config"
	"problem-search/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	rootCtx := context.Background()

	startupCtx, startupCancel := context.WithTimeout(rootCtx, 10*time.Second)
	defer startupCancel()

	store, err := storage.NewPostgresStore(startupCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	client, err := leetcode.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	if err := client.InitSession(); err != nil {
		log.Fatal(err)
	}

	ingester := ingest.NewLeetCodeIngester(client, store)

	skip := 0
	limit := 20
	page := 0

	for {
		log.Printf("starting page=%d skip=%d limit=%d", page+1, skip, limit)
		pageCtx, pageCancel := context.WithTimeout(rootCtx, 6*time.Minute)
		count, err := ingester.IngestPage(pageCtx, skip, limit)
		pageCancel()
		if err != nil {
			log.Fatalf("ingest page %d failed: %v", page+1, err)
		}

		if count == 0 {
			log.Println("no more questions returned, stopping")
			break
		}

		log.Printf("page=%d fetched=%d", page+1, count)

		if count < limit {
			log.Println("last partial page reached, stopping")
			break
		}
		page++
		skip += limit
	}
}
