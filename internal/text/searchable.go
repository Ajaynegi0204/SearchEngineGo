package text

import (
	"strings"

	"problem-search/internal/models"
)

// SearchableProblem creates the same text representation for indexing and querying.
func SearchableProblem(problem models.Problem) string {
	return strings.Join([]string{
		problem.Title,
		strings.Join(problem.Tags, " "),
		problem.StatementText,
	}, " ")
}
