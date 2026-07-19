package text

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/unicode/norm"
)

// Keep this deliberately conservative. Negation and constraint words such as
// "not", "no", "only", and "without" can change a problem's meaning.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
	"be": {}, "by": {}, "do": {}, "does": {}, "for": {}, "from": {},
	"has": {}, "have": {}, "if": {}, "in": {}, "into": {}, "is": {},
	"it": {}, "its": {}, "of": {}, "on": {}, "or": {}, "that": {},
	"the": {}, "their": {}, "then": {}, "there": {}, "these": {},
	"they": {}, "this": {}, "those": {}, "to": {}, "was": {}, "were": {},
	"when": {}, "where": {}, "which": {}, "who": {}, "with": {}, "you": {},
	"your": {},
}

// Retain letters and digits. The suffix preserves common technical tokens
// such as C++, C#, and 2D.
var tokenPattern = regexp.MustCompile(`[\pL\pN]+(?:\+\+|#)?`)

const TokenizerVersion = "v1"

func Tokenize(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	normalized := norm.NFKC.String(html.UnescapeString(value))
	normalized = strings.ToLower(normalized)

	matches := tokenPattern.FindAllString(normalized, -1)
	if len(matches) == 0 {
		return nil
	}

	tokens := make([]string, 0, len(matches))
	for _, token := range matches {
		if _, isStopWord := stopWords[token]; isStopWord {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens
}
