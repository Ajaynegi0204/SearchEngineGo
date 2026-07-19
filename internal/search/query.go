package search

import (
	"errors"
	"strings"
)

const maxQueryLength = 1000

var ErrInvalidQuery = errors.New("query cannot be empty")

func normalizeQuery(query string) (string, error) {
	normalizedQuery := strings.Join(strings.Fields(query), " ")
	if normalizedQuery == "" {
		return "", ErrInvalidQuery
	}

	queryRunes := []rune(normalizedQuery)
	if len(queryRunes) > maxQueryLength {
		return string(queryRunes[:maxQueryLength]), nil
	}

	return normalizedQuery, nil
}
