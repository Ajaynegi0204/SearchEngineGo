package embedding

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

const embedModel = "embed-v4.0"

type CohereClient struct {
	apiKey     string
	httpClient *http.Client
}

type embedRequest struct {
	Model          string   `json:"model"`
	InputType      string   `json:"input_type"`
	EmbeddingTypes []string `json:"embedding_types"`
	Texts          []string `json:"texts"`
}

type embedResponse struct {
	Embeddings struct {
		Float [][]float32 `json:"float"`
	} `json:"embeddings"`
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

func (c *CohereClient) EmbedTexts(
	ctx context.Context,
	texts []string,
	inputType string,
) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}
	if len(texts) > 96 {
		return nil, fmt.Errorf("Cohere accepts at most 96 texts per request")
	}

	requestBody := embedRequest{
		Model:          embedModel,
		InputType:      inputType,
		EmbeddingTypes: []string{"float"},
		Texts:          texts,
	}

	encodedBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.cohere.com/v2/embed",
		bytes.NewReader(encodedBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send embedding request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if attempt == 3 {
				return nil, fmt.Errorf("embedding request rate limit exceeded after %d attempts", attempt)
			}

			wait := time.Duration(attempt) * time.Minute
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}

			requestBody, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("reset embedding request body: %w", err)
			}
			req.Body = requestBody
			continue
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
			resp.Body.Close()
			if readErr != nil {
				return nil, fmt.Errorf("embedding request returned status %d and response could not be read: %w", resp.StatusCode, readErr)
			}
			return nil, fmt.Errorf("embedding request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var response embedResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode embedding response: %w", err)
		}
		resp.Body.Close()

		if len(response.Embeddings.Float) != len(texts) {
			return nil, fmt.Errorf("Cohere returned %d embeddings for %d texts", len(response.Embeddings.Float), len(texts))
		}

		return response.Embeddings.Float, nil
	}

	return nil, fmt.Errorf("embedding request failed")
}
