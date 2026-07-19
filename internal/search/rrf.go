package search

import "sort"

const defaultRRFConstant = 60

type RankedProblem struct {
	ID            int64
	BM25Rank      int
	EmbeddingRank int
	RRFScore      float64
}

func CombineWithRRF(
	bm25Results []int64,
	embeddingResults []int64,
	resultLimit int,
) []RankedProblem {
	if resultLimit <= 0 {
		return nil
	}

	byID := make(map[int64]*RankedProblem, len(bm25Results)+len(embeddingResults))

	for rank, id := range bm25Results {
		problem := getOrCreateRankedProblem(byID, id)
		problem.BM25Rank = rank + 1
		problem.RRFScore += 1.0 / float64(defaultRRFConstant+rank+1)
	}

	for rank, id := range embeddingResults {
		problem := getOrCreateRankedProblem(byID, id)
		problem.EmbeddingRank = rank + 1
		problem.RRFScore += 1.0 / float64(defaultRRFConstant+rank+1)
	}

	combined := make([]RankedProblem, 0, len(byID))
	for _, problem := range byID {
		combined = append(combined, *problem)
	}

	sort.Slice(combined, func(i, j int) bool {
		if combined[i].RRFScore == combined[j].RRFScore {
			return combined[i].ID < combined[j].ID
		}
		return combined[i].RRFScore > combined[j].RRFScore
	})

	if resultLimit > len(combined) {
		resultLimit = len(combined)
	}
	return combined[:resultLimit]
}

func getOrCreateRankedProblem(byID map[int64]*RankedProblem, id int64) *RankedProblem {
	if problem, ok := byID[id]; ok {
		return problem
	}

	problem := &RankedProblem{ID: id}
	byID[id] = problem
	return problem
}
