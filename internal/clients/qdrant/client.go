package qdrant

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/qdrant/go-client/qdrant"

	"problem-search/internal/retrieval"
)

type Client struct {
	client *qdrant.Client
}

type SparsePoint struct {
	ID     uint64
	Vector retrieval.SparseVector
}

type HybridPoint struct {
	ID         uint64
	BM25Vector retrieval.SparseVector
	Embedding  []float32
}

type SearchResult struct {
	ID    uint64
	Score float32
}

func NewClient(endpoint, apiKey string) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return nil, fmt.Errorf("parse qdrant endpoint: %w", err)
	}
	if parsedURL.Hostname() == "" {
		return nil, fmt.Errorf("qdrant endpoint must include a host")
	}

	port := 6334
	if parsedURL.Port() != "" {
		port, err = strconv.Atoi(parsedURL.Port())
		if err != nil {
			return nil, fmt.Errorf("parse qdrant port: %w", err)
		}
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:     parsedURL.Hostname(),
		Port:     port,
		APIKey:   apiKey,
		UseTLS:   parsedURL.Scheme == "https",
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("create qdrant client: %w", err)
	}

	return &Client{client: client}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("qdrant health check: %w", err)
	}
	return nil
}

func (c *Client) CreateCollection(ctx context.Context, collectionName string) error {
	if strings.TrimSpace(collectionName) == "" {
		return fmt.Errorf("collection name is required")
	}

	request := &qdrant.CreateCollection{
		CollectionName: collectionName,
		SparseVectorsConfig: qdrant.NewSparseVectorsConfig(
			map[string]*qdrant.SparseVectorParams{
				"bm25": {},
			},
		),
	}

	if err := c.client.CreateCollection(ctx, request); err != nil {
		return fmt.Errorf("create qdrant collection: %w", err)
	}
	return nil
}

func (c *Client) EnsureHybridCollection(ctx context.Context, collectionName string, embeddingDimension uint64) error {
	if strings.TrimSpace(collectionName) == "" {
		return fmt.Errorf("collection name is required")
	}
	if embeddingDimension == 0 {
		return fmt.Errorf("embedding dimension must be greater than zero")
	}

	exists, err := c.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("check hybrid collection: %w", err)
	}
	if exists {
		return nil
	}

	request := &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfigMap(
			map[string]*qdrant.VectorParams{
				"embedding": {
					Size:     embeddingDimension,
					Distance: qdrant.Distance_Cosine,
				},
			},
		),
		SparseVectorsConfig: qdrant.NewSparseVectorsConfig(
			map[string]*qdrant.SparseVectorParams{
				"bm25": {},
			},
		),
	}

	if err := c.client.CreateCollection(ctx, request); err != nil {
		return fmt.Errorf("create hybrid collection: %w", err)
	}
	return nil
}

func (c *Client) UpsertSparseVectors(
	ctx context.Context,
	collectionName string,
	points []SparsePoint,
) error {
	if len(points) == 0 {
		return nil
	}

	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, point := range points {
		if len(point.Vector.Indices) != len(point.Vector.Values) {
			return fmt.Errorf("indices and values length mismatch for point %d", point.ID)
		}

		qdrantPoints = append(qdrantPoints, &qdrant.PointStruct{
			Id: qdrant.NewIDNum(point.ID),
			Vectors: qdrant.NewVectorsMap(
				map[string]*qdrant.Vector{
					"bm25": qdrant.NewVectorSparse(point.Vector.Indices, point.Vector.Values),
				},
			),
		})
	}

	request := &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Wait:           qdrant.PtrOf(true),
		Points:         qdrantPoints,
	}

	if _, err := c.client.Upsert(ctx, request); err != nil {
		return fmt.Errorf("upsert sparse vectors: %w", err)
	}
	return nil
}

func (c *Client) UpsertHybridPoints(
	ctx context.Context,
	collectionName string,
	points []HybridPoint,
) error {
	if len(points) == 0 {
		return nil
	}

	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, point := range points {
		if len(point.BM25Vector.Indices) != len(point.BM25Vector.Values) {
			return fmt.Errorf("BM25 indices and values length mismatch for point %d", point.ID)
		}
		if len(point.Embedding) == 0 {
			return fmt.Errorf("embedding cannot be empty for point %d", point.ID)
		}

		qdrantPoints = append(qdrantPoints, &qdrant.PointStruct{
			Id: qdrant.NewIDNum(point.ID),
			Vectors: qdrant.NewVectorsMap(
				map[string]*qdrant.Vector{
					"bm25":      qdrant.NewVectorSparse(point.BM25Vector.Indices, point.BM25Vector.Values),
					"embedding": qdrant.NewVectorDense(point.Embedding),
				},
			),
		})
	}

	request := &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Wait:           qdrant.PtrOf(true),
		Points:         qdrantPoints,
	}

	if _, err := c.client.Upsert(ctx, request); err != nil {
		return fmt.Errorf("upsert hybrid points: %w", err)
	}
	return nil
}

func (c *Client) SearchSparseVectors(
	ctx context.Context,
	collectionName string,
	vector retrieval.SparseVector,
	limit uint64,
) ([]SearchResult, error) {
	if limit == 0 {
		return nil, fmt.Errorf("search limit must be greater than zero")
	}
	if len(vector.Indices) != len(vector.Values) {
		return nil, fmt.Errorf("indices and values length mismatch")
	}

	points, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQuerySparse(vector.Indices, vector.Values),
		Using:          qdrant.PtrOf("bm25"),
		Limit:          qdrant.PtrOf(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search sparse vectors: %w", err)
	}

	results := make([]SearchResult, 0, len(points))
	for _, point := range points {
		results = append(results, SearchResult{
			ID:    point.GetId().GetNum(),
			Score: point.GetScore(),
		})
	}

	return results, nil
}

func (c *Client) SearchDenseVectors(
	ctx context.Context,
	collectionName string,
	vector []float32,
	limit uint64,
) ([]SearchResult, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("dense search vector cannot be empty")
	}
	if limit == 0 {
		return nil, fmt.Errorf("search limit must be greater than zero")
	}

	points, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQueryDense(vector),
		Using:          qdrant.PtrOf("embedding"),
		Limit:          qdrant.PtrOf(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search dense vectors: %w", err)
	}

	results := make([]SearchResult, 0, len(points))
	for _, point := range points {
		results = append(results, SearchResult{
			ID:    point.GetId().GetNum(),
			Score: point.GetScore(),
		})
	}

	return results, nil
}
