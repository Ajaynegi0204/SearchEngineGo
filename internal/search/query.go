package search

import (
	"errors"
	"strings"
)

var ErrInvalidQuery = errors.New("query cannot be empty")

func normalizeQuery(query string) (string, error) {
	normalizedQuery := strings.Join(strings.Fields(query), " ")
	if normalizedQuery == "" {
		return "", ErrInvalidQuery
	}
	return normalizedQuery, nil
}
