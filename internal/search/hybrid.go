package search

import (
	"context"
	"fmt"

	"problem-search/internal/clients/embedding"
	"problem-search/internal/clients/qdrant"
	"problem-search/internal/clients/rerank"
	"problem-search/internal/models"
	"problem-search/internal/retrieval"
	"problem-search/internal/storage"
	"problem-search/internal/text"
)

type HybridSearchResult struct {
	Problem       models.Problem
	BM25Rank      int
	EmbeddingRank int
	RRFScore      float64
	RerankScore   float64
}

type HybridEngine struct {
	store          *storage.PostgresStore
	vectorStore    *qdrant.Client
	embedding      *embedding.CohereClient
	reranker       *rerank.CohereClient
	bm25Index      *retrieval.BM25Index
	collectionName string
	candidateLimit uint64
	resultLimit    int
}

func NewHybridEngine(
	store *storage.PostgresStore,
	vectorStore *qdrant.Client,
	embeddingClient *embedding.CohereClient,
	rerankerClient *rerank.CohereClient,
	bm25Index *retrieval.BM25Index,
	collectionName string,
	candidateLimit uint64,
	resultLimit int,
) (*HybridEngine, error) {
	if store == nil || vectorStore == nil || embeddingClient == nil || rerankerClient == nil || bm25Index == nil {
		return nil, fmt.Errorf("all hybrid search dependencies are required")
	}
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	if candidateLimit == 0 {
		return nil, fmt.Errorf("candidate limit must be greater than zero")
	}
	if resultLimit <= 0 {
		return nil, fmt.Errorf("result limit must be greater than zero")
	}

	return &HybridEngine{
		store:          store,
		vectorStore:    vectorStore,
		embedding:      embeddingClient,
		reranker:       rerankerClient,
		bm25Index:      bm25Index,
		collectionName: collectionName,
		candidateLimit: candidateLimit,
		resultLimit:    resultLimit,
	}, nil
}

func (e *HybridEngine) Search(ctx context.Context, query string) ([]HybridSearchResult, error) {
	normalizedQuery, err := normalizeQuery(query)
	if err != nil {
		return nil, err
	}
	query = normalizedQuery

	bm25IDs, err := e.searchBM25(ctx, query)
	if err != nil {
		return nil, err
	}

	embeddingVectors, err := e.embedding.EmbedTexts(ctx, []string{query}, "search_query")
	if err != nil {
		return nil, fmt.Errorf("embed search query: %w", err)
	}

	if len(embeddingVectors) != 1 {
		return nil, fmt.Errorf("Cohere returned %d query embeddings, expected 1", len(embeddingVectors))
	}

	embeddingResults, err := e.vectorStore.SearchDenseVectors(
		ctx,
		e.collectionName,
		embeddingVectors[0],
		e.candidateLimit,
	)
	if err != nil {
		return nil, err
	}

	embeddingIDs := make([]int64, 0, len(embeddingResults))
	for _, result := range embeddingResults {
		embeddingIDs = append(embeddingIDs, int64(result.ID))
	}

	rankedProblems := CombineWithRRF(bm25IDs, embeddingIDs, int(e.candidateLimit))
	problemIDs := make([]int64, 0, len(rankedProblems))
	for _, rankedProblem := range rankedProblems {
		problemIDs = append(problemIDs, rankedProblem.ID)
	}

	problems, err := e.store.GetProblemsByIDs(ctx, problemIDs)
	if err != nil {
		return nil, fmt.Errorf("load hybrid search results: %w", err)
	}

	candidates := make([]HybridSearchResult, 0, len(rankedProblems))
	for _, rankedProblem := range rankedProblems {
		matchedProblem, ok := problems[rankedProblem.ID]
		if !ok {
			continue
		}

		candidates = append(candidates, HybridSearchResult{
			Problem:       matchedProblem,
			BM25Rank:      rankedProblem.BM25Rank,
			EmbeddingRank: rankedProblem.EmbeddingRank,
			RRFScore:      rankedProblem.RRFScore,
		})
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	documents := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		documents = append(documents, text.SearchableProblem(candidate.Problem))
	}

	reranked, err := e.reranker.Rerank(ctx, query, documents, min(e.resultLimit, len(documents)))
	if err != nil {
		return nil, fmt.Errorf("rerank search results: %w", err)
	}

	results := make([]HybridSearchResult, 0, len(reranked))
	for _, ranked := range reranked {
		result := candidates[ranked.Index]
		result.RerankScore = ranked.RelevanceScore
		results = append(results, result)
	}

	return results, nil
}

func (e *HybridEngine) searchBM25(ctx context.Context, query string) ([]int64, error) {
	queryVector, err := e.bm25Index.BuildQueryVector(query)
	if err != nil {
		return nil, fmt.Errorf("build BM25 query vector: %w", err)
	}
	if len(queryVector.Indices) == 0 {
		return nil, nil
	}

	results, err := e.vectorStore.SearchSparseVectors(
		ctx,
		e.collectionName,
		queryVector,
		e.candidateLimit,
	)
	if err != nil {
		return nil, err
	}

	problemIDs := make([]int64, 0, len(results))
	for _, result := range results {
		problemIDs = append(problemIDs, int64(result.ID))
	}
	return problemIDs, nil
}
