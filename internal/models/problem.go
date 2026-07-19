package models

type Problem struct {
	ID            int64
	Platform      string
	ExternalID    string
	Slug          string
	Title         string
	URL           string
	Difficulty    string
	Tags          []string
	StatementText string
}
