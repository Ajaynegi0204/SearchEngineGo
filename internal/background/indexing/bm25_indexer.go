package indexing

import (
	"fmt"
	"strconv"

	"problem-search/internal/models"
	"problem-search/internal/retrieval"
	"problem-search/internal/text"
)

type ProblemVector struct {
	Problem models.Problem
	Vector  retrieval.SparseVector
}

type BuildResult struct {
	Index          *retrieval.BM25Index
	ProblemVectors []ProblemVector
}

type BM25Indexer struct {
	k1 float64
	b  float64
}

func NewBM25Indexer(k1, b float64) (*BM25Indexer, error) {
	if _, err := retrieval.NewBM25Index(k1, b); err != nil {
		return nil, err
	}

	return &BM25Indexer{
		k1: k1,
		b:  b,
	}, nil
}

func (i *BM25Indexer) Build(problems []models.Problem) (*BuildResult, error) {
	if len(problems) == 0 {
		return nil, fmt.Errorf("cannot build vectors without problems")
	}

	bm25, err := retrieval.NewBM25Index(i.k1, i.b)
	if err != nil {
		return nil, fmt.Errorf("create bm25 index: %w", err)
	}

	for _, p := range problems {
		documentID := strconv.FormatInt(p.ID, 10)
		if err := bm25.AddDocument(documentID, text.SearchableProblem(p)); err != nil {
			return nil, fmt.Errorf("add problem %d to bm25 index: %w", p.ID, err)
		}
	}

	if err := bm25.FinalizeBuild(); err != nil {
		return nil, fmt.Errorf("finalize bm25 index: %w", err)
	}

	problemVectors := make([]ProblemVector, 0, len(problems))
	for _, p := range problems {
		documentID := strconv.FormatInt(p.ID, 10)
		vector, err := bm25.BuildDocumentVector(documentID)
		if err != nil {
			return nil, fmt.Errorf("build vector for problem %d: %w", p.ID, err)
		}

		problemVectors = append(problemVectors, ProblemVector{
			Problem: p,
			Vector:  vector,
		})
	}

	return &BuildResult{
		Index:          bm25,
		ProblemVectors: problemVectors,
	}, nil
}
