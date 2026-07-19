package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"problem-search/internal/search"
)

type QueryHandler struct {
	searchEngine *search.HybridEngine
}

type queryRequest struct {
	Query string `json:"query"`
}

type queryResponse struct {
	Results []problemResult `json:"results"`
}

type problemResult struct {
	ID         int64    `json:"id"`
	Title      string   `json:"title"`
	Slug       string   `json:"slug"`
	URL        string   `json:"url"`
	Difficulty string   `json:"difficulty"`
	Tags       []string `json:"tags"`
}

func NewQueryHandler(searchEngine *search.HybridEngine) *QueryHandler {
	return &QueryHandler{searchEngine: searchEngine}
}

func (h *QueryHandler) Query(w http.ResponseWriter, r *http.Request) {
	var request queryRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	results, err := h.searchEngine.Search(r.Context(), request.Query)
	if err != nil {
		if errors.Is(err, search.ErrInvalidQuery) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("search failed: %v", err)
		writeError(w, http.StatusInternalServerError, "search is temporarily unavailable")
		return
	}

	responseResults := make([]problemResult, 0, len(results))
	for _, result := range results {
		responseResults = append(responseResults, problemResult{
			ID:         result.Problem.ID,
			Title:      result.Problem.Title,
			Slug:       result.Problem.Slug,
			URL:        result.Problem.URL,
			Difficulty: result.Problem.Difficulty,
			Tags:       result.Problem.Tags,
		})
	}

	writeJSON(w, http.StatusOK, queryResponse{Results: responseResults})
}
