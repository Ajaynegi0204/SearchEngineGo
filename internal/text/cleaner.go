package text

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func HTMLToText(rawHTML string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))

	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}

	doc.Find("script, style").Remove()
	cleaned := doc.Text()
	cleaned = normalizeWhitespace(cleaned)

	return cleaned, nil
}

func normalizeWhitespace(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")

	blankLines := regexp.MustCompile(`\n\s*\n+`)
	spaces := regexp.MustCompile(`[ \t]+`)

	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(spaces.ReplaceAllString(line, " "))
	}

	value = strings.Join(lines, "\n")
	value = blankLines.ReplaceAllString(value, "\n\n")

	return strings.TrimSpace(value)
}
