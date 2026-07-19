package retrieval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	searchtext "problem-search/internal/text"
)

type SparseVector struct {
	Indices []uint32
	Values  []float32
}

type savedBM25Index struct {
	K1                    float64
	B                     float64
	TotalDocuments        int
	AverageDocumentLength float64
	DocumentLengths       map[string]int
	TermFrequencies       map[string]map[string]int
	DocumentFrequencies   map[string]int
	InverseDocumentFreqs  map[string]float64
	Vocabulary            map[string]uint32
	TokenizerVersion      string
}

type BM25Index struct {
	K1 float64
	B  float64

	totalDocuments        int
	averageDocumentLength float64
	documentLengths       map[string]int
	termFrequencies       map[string]map[string]int
	documentFrequencies   map[string]int
	inverseDocumentFreqs  map[string]float64
	vocabulary            map[string]uint32
	finalized             bool
}

func NewBM25Index(k1, b float64) (*BM25Index, error) {
	if k1 < 0 {
		return nil, fmt.Errorf("k1 must be non-negative")
	}
	if b < 0 || b > 1 {
		return nil, fmt.Errorf("b must be between 0 and 1")
	}

	return &BM25Index{
		K1:                   k1,
		B:                    b,
		documentLengths:      make(map[string]int),            // document length for each document
		termFrequencies:      make(map[string]map[string]int), // term frequency for each document
		documentFrequencies:  make(map[string]int),            // no. of document containing the term
		inverseDocumentFreqs: make(map[string]float64),        // IDF for each term
		vocabulary:           make(map[string]uint32),         // vocabulary for the index
	}, nil
}

func (i *BM25Index) AddDocument(documentID, value string) error {
	if documentID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}
	if _, exists := i.termFrequencies[documentID]; exists {
		return fmt.Errorf("document already added: %s", documentID)
	}

	tokens := searchtext.Tokenize(value)
	termCounts := make(map[string]int)
	seenTerms := make(map[string]struct{})

	for _, token := range tokens {
		termCounts[token]++
		if _, seen := seenTerms[token]; !seen {
			i.documentFrequencies[token]++
			seenTerms[token] = struct{}{}
		}
	}

	i.termFrequencies[documentID] = termCounts
	i.documentLengths[documentID] = len(tokens)
	i.totalDocuments++
	i.finalized = false

	return nil
}

func (i *BM25Index) FinalizeBuild() error {
	if i.totalDocuments == 0 {
		return fmt.Errorf("no documents have been added")
	}

	totalLength := 0
	for _, documentLength := range i.documentLengths {
		totalLength += documentLength
	}
	i.averageDocumentLength = float64(totalLength) / float64(i.totalDocuments)

	terms := make([]string, 0, len(i.documentFrequencies))
	for term := range i.documentFrequencies {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	i.vocabulary = make(map[string]uint32, len(terms))
	i.inverseDocumentFreqs = make(map[string]float64, len(terms))

	for termID, term := range terms {
		i.vocabulary[term] = uint32(termID)

		documentFrequency := i.documentFrequencies[term]
		numerator := float64(i.totalDocuments-documentFrequency) + 0.5
		denominator := float64(documentFrequency) + 0.5
		i.inverseDocumentFreqs[term] = math.Log(1.0 + numerator/denominator)
	}

	i.finalized = true
	return nil
}

func (i *BM25Index) BuildDocumentVector(documentID string) (SparseVector, error) {
	if err := i.ensureFinalized(); err != nil {
		return SparseVector{}, err
	}

	termCounts, ok := i.termFrequencies[documentID]
	if !ok {
		return SparseVector{}, fmt.Errorf("unknown document ID: %s", documentID)
	}

	if i.averageDocumentLength == 0 {
		return SparseVector{}, nil
	}

	documentLength := i.documentLengths[documentID]
	lengthNormalizer := 1.0 - i.B + i.B*(float64(documentLength)/i.averageDocumentLength)

	type weightedTerm struct {
		id     uint32
		weight float32
	}

	weights := make([]weightedTerm, 0, len(termCounts))
	for term, count := range termCounts {
		termID := i.vocabulary[term]
		idf := i.inverseDocumentFreqs[term]
		normalizedTF := (float64(count) * (i.K1 + 1.0)) /
			(float64(count) + i.K1*lengthNormalizer)

		weights = append(weights, weightedTerm{
			id:     termID,
			weight: float32(idf * normalizedTF),
		})
	}

	sort.Slice(weights, func(a, b int) bool {
		return weights[a].id < weights[b].id
	})

	vector := SparseVector{
		Indices: make([]uint32, 0, len(weights)),
		Values:  make([]float32, 0, len(weights)),
	}

	for _, weighted := range weights {
		vector.Indices = append(vector.Indices, weighted.id)
		vector.Values = append(vector.Values, weighted.weight)
	}

	return vector, nil
}

func (i *BM25Index) BuildAllDocumentVectors() (map[string]SparseVector, error) {
	if err := i.ensureFinalized(); err != nil {
		return nil, err
	}

	vectors := make(map[string]SparseVector, len(i.termFrequencies))
	for documentID := range i.termFrequencies {
		vector, err := i.BuildDocumentVector(documentID)
		if err != nil {
			return nil, err
		}
		vectors[documentID] = vector
	}

	return vectors, nil
}

func (i *BM25Index) BuildQueryVector(query string) (SparseVector, error) {
	if err := i.ensureFinalized(); err != nil {
		return SparseVector{}, err
	}

	seen := make(map[uint32]struct{})
	for _, token := range searchtext.Tokenize(query) {
		termID, ok := i.vocabulary[token]
		if !ok {
			continue
		}
		seen[termID] = struct{}{}
	}

	indices := make([]uint32, 0, len(seen))
	for termID := range seen {
		indices = append(indices, termID)
	}
	sort.Slice(indices, func(a, b int) bool {
		return indices[a] < indices[b]
	})

	vector := SparseVector{
		Indices: indices,
		Values:  make([]float32, len(indices)),
	}
	for idx := range vector.Values {
		vector.Values[idx] = 1.0
	}

	return vector, nil
}

func (i *BM25Index) Save(path string) error {
	if err := i.ensureFinalized(); err != nil {
		return err
	}

	savedIndex := savedBM25Index{
		K1:                    i.K1,
		B:                     i.B,
		TotalDocuments:        i.totalDocuments,
		AverageDocumentLength: i.averageDocumentLength,
		DocumentLengths:       i.documentLengths,
		TermFrequencies:       i.termFrequencies,
		DocumentFrequencies:   i.documentFrequencies,
		InverseDocumentFreqs:  i.inverseDocumentFreqs,
		Vocabulary:            i.vocabulary,
		TokenizerVersion:      searchtext.TokenizerVersion,
	}

	data, err := json.MarshalIndent(savedIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("encode bm25 index: %w", err)
	}

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create bm25 index directory: %w", err)
	}

	temporaryPath := path + ".tmp"
	if err := os.WriteFile(temporaryPath, data, 0o644); err != nil {
		return fmt.Errorf("write bm25 index: %w", err)
	}

	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("save bm25 index: %w", err)
	}

	return nil
}

func LoadBM25Index(path string) (*BM25Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bm25 index: %w", err)
	}

	var savedIndex savedBM25Index
	if err := json.Unmarshal(data, &savedIndex); err != nil {
		return nil, fmt.Errorf("decode bm25 index: %w", err)
	}
	if savedIndex.TokenizerVersion != searchtext.TokenizerVersion {
		return nil, fmt.Errorf("unsupported tokenizer version: %s", savedIndex.TokenizerVersion)
	}
	if savedIndex.TotalDocuments == 0 {
		return nil, fmt.Errorf("bm25 index contains no documents")
	}

	return &BM25Index{
		K1:                    savedIndex.K1,
		B:                     savedIndex.B,
		totalDocuments:        savedIndex.TotalDocuments,
		averageDocumentLength: savedIndex.AverageDocumentLength,
		documentLengths:       savedIndex.DocumentLengths,
		termFrequencies:       savedIndex.TermFrequencies,
		documentFrequencies:   savedIndex.DocumentFrequencies,
		inverseDocumentFreqs:  savedIndex.InverseDocumentFreqs,
		vocabulary:            savedIndex.Vocabulary,
		finalized:             true,
	}, nil
}

func (i *BM25Index) TotalDocuments() int {
	return i.totalDocuments
}

func (i *BM25Index) AverageDocumentLength() float64 {
	return i.averageDocumentLength
}

func (i *BM25Index) Vocabulary() map[string]uint32 {
	copyMap := make(map[string]uint32, len(i.vocabulary))
	for term, id := range i.vocabulary {
		copyMap[term] = id
	}
	return copyMap
}

func (i *BM25Index) IDF(term string) (float64, error) {
	if err := i.ensureFinalized(); err != nil {
		return 0, err
	}
	return i.inverseDocumentFreqs[term], nil
}

func (i *BM25Index) ensureFinalized() error {
	if !i.finalized {
		return fmt.Errorf("bm25 index has not been finalized")
	}
	return nil
}
