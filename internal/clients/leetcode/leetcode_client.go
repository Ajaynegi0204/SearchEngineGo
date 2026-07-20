package leetcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	csrfToken  string
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		},
		baseURL: "https://leetcode.com/graphql",
	}, nil
}

func (c *Client) InitSession() error {
	sessionURL := "https://leetcode.com/problems/two-sum/"

	req, err := http.NewRequest(
		http.MethodGet,
		sessionURL,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create session request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("initialize leetcode session: %w", err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	leetcodeURL, err := url.Parse(sessionURL)
	if err != nil {
		return fmt.Errorf("parse leetcode url: %w", err)
	}

	for _, cookie := range c.httpClient.Jar.Cookies(leetcodeURL) {
		if cookie.Name == "csrftoken" {
			c.csrfToken = cookie.Value
			return nil
		}
	}

	return fmt.Errorf("csrftoken cookie not found")
}

type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type GraphQLResponse struct {
	Data struct {
		Question Question `json:"question"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

type Question struct {
	QuestionID string `json:"questionId"`
	Title      string `json:"title"`
	TitleSlug  string `json:"titleSlug"`
	Content    string `json:"content"`
	Difficulty string `json:"difficulty"`
	TopicTags  []Tag  `json:"topicTags"`
}

type Tag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type QuestionListResponse struct {
	Data struct {
		ProblemsetQuestionList ProblemsetQuestionList `json:"problemsetQuestionListV2"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type ProblemsetQuestionList struct {
	Questions []QuestionSummary `json:"questions"`
}

type QuestionSummary struct {
	TitleSlug string `json:"titleSlug"`
	PaidOnly  bool   `json:"paidOnly"`
}

const questionListQuery = `
  query problemsetQuestionList($categorySlug: String, $limit: Int, $skip:
  Int, $filters: QuestionFilterInput) {
    problemsetQuestionListV2(
      categorySlug: $categorySlug
      limit: $limit
      skip: $skip
      filters: $filters
    ) {
      questions {
        titleSlug
        paidOnly
      }
    }
  }
  `

const questionQuery = `
  query questionData($titleSlug: String!) {
    question(titleSlug: $titleSlug) {
      questionId
      title
      titleSlug
      content
      difficulty
      topicTags {
        name
        slug
      }
    }
  }
  `

func (c *Client) doGraphQL(
	query string,
	variables map[string]any,
	referer string,
	target any,
) error {
	requestBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(requestBody); err != nil {
		return fmt.Errorf("encode request body: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		c.baseURL,
		&buf,
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://leetcode.com")
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	if c.csrfToken != "" {
		req.Header.Set("x-csrftoken", c.csrfToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		if _, err := errorBody.ReadFrom(resp.Body); err != nil {
			return fmt.Errorf("leetcode returned status %d and failed to read error body: %w", resp.StatusCode, err)
		}

		return fmt.Errorf("leetcode returned status %d: %s", resp.StatusCode, errorBody.String())
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

//MARK: LeetCode Fetch

func (c *Client) FetchQuestionDetails(titleSlug string) (*Question, error) {
	var graphQLResponse GraphQLResponse

	err := c.doGraphQL(
		questionQuery,
		map[string]any{
			"titleSlug": titleSlug,
		},
		"https://leetcode.com/problems/"+titleSlug+"/",
		&graphQLResponse,
	)
	if err != nil {
		return nil, err
	}

	if len(graphQLResponse.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s",
			graphQLResponse.Errors[0].Message)
	}

	return &graphQLResponse.Data.Question, nil
}

func (c *Client) FetchQuestionList(skip int, limit int) ([]QuestionSummary, error) {
	var response QuestionListResponse

	err := c.doGraphQL(
		questionListQuery,
		map[string]any{
			"categorySlug": "",
			"skip":         skip,
			"limit":        limit,
			"filters": map[string]any{
				"filterCombineType": "ALL",
			},
		},
		"https://leetcode.com/problemset/",
		&response,
	)

	if err != nil {
		return nil, err
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", response.Errors[0].Message)
	}

	return response.Data.ProblemsetQuestionList.Questions, nil
}
