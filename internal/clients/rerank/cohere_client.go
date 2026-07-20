package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const rerankModel = "rerank-v4.0-pro"

type CohereClient struct {
	apiKey     string
	httpClient *http.Client
}

type Result struct {
	Index          int
	RelevanceScore float64
}

type rerankRequest struct {
	Model          string   `json:"model"`
	Query          string   `json:"query"`
	Documents      []string `json:"documents"`
	TopN           int      `json:"top_n"`
	ReturnDocument bool     `json:"return_documents"`
}

type rerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

func NewCohereClient(apiKey string) (*CohereClient, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("Cohere API key is required")
	}

	return &CohereClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (c *CohereClient) Rerank(
	ctx context.Context,
	query string,
	documents []string,
	topN int,
) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("rerank query cannot be empty")
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("rerank documents cannot be empty")
	}
	if len(documents) > 1000 {
		return nil, fmt.Errorf("rerank accepts at most 1000 documents")
	}
	if topN <= 0 || topN > len(documents) {
		return nil, fmt.Errorf("rerank topN must be between 1 and %d", len(documents))
	}

	requestBody := rerankRequest{
		Model:          rerankModel,
		Query:          query,
		Documents:      documents,
		TopN:           topN,
		ReturnDocument: false,
	}

	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode rerank request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.cohere.com/v2/rerank",
		bytes.NewReader(encodedBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create rerank request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send rerank request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		if readErr != nil {
			return nil, fmt.Errorf("rerank request returned status %d and response could not be read: %w", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("rerank request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}

	results := make([]Result, 0, len(response.Results))
	for _, result := range response.Results {
		if result.Index < 0 || result.Index >= len(documents) {
			return nil, fmt.Errorf("rerank returned invalid document index %d", result.Index)
		}
		results = append(results, Result{
			Index:          result.Index,
			RelevanceScore: result.RelevanceScore,
		})
	}

	return results, nil
}
