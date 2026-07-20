package leetcode

import "problem-search/internal/models"

func ToProblem(question *Question, statementText string) models.Problem {
	tags := make([]string, 0, len(question.TopicTags))
	for _, tag := range question.TopicTags {
		tags = append(tags, tag.Name)
	}

	return models.Problem{
		Platform:      "leetcode",
		ExternalID:    question.QuestionID,
		Slug:          question.TitleSlug,
		Title:         question.Title,
		URL:           "https://leetcode.com/problems/" + question.TitleSlug + "/",
		Difficulty:    question.Difficulty,
		Tags:          tags,
		StatementText: statementText,
	}
}
