package ingest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"problem-search/internal/background/leetcode"
	"problem-search/internal/models"
	"problem-search/internal/storage"
	"problem-search/internal/text"
)

type LeetCodeIngester struct {
	client       *leetcode.Client
	requestDelay time.Duration
	dbStore      *storage.PostgresStore
	workerCount  int
}

var ErrPaidProblem = errors.New("paid only problem")

func NewLeetCodeIngester(client *leetcode.Client, dbStore *storage.PostgresStore) *LeetCodeIngester {
	return &LeetCodeIngester{
		client:       client,
		requestDelay: 300 * time.Millisecond,
		dbStore:      dbStore,
		workerCount:  5,
	}
}

func (i *LeetCodeIngester) IngestPage(ctx context.Context, skip int, limit int) (int, error) {
	list, err := i.client.FetchQuestionList(skip, limit)
	if err != nil {
		return 0, err
	}
	log.Printf("fetched page summaries count=%d", len(list))

	jobs := make(chan leetcode.QuestionSummary, i.workerCount*2)
	results := make(chan models.Problem, i.workerCount*2)

	var wg sync.WaitGroup
	for worker := 0; worker < i.workerCount; worker++ {
		wg.Add(1)
		go i.worker(ctx, jobs, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for _, summary := range list {
		select {
		case jobs <- summary:
		case <-ctx.Done():
			close(jobs)
			return 0, ctx.Err()
		}
	}

	close(jobs)

	problems := make([]models.Problem, 0, len(list))
	for problemModel := range results {
		problems = append(problems, problemModel)
	}
	log.Printf("writing problems to db count=%d", len(problems))

	err = i.dbStore.UpsertProblems(ctx, problems)
	if err != nil {
		return 0, fmt.Errorf("error inserting to db: %w", err)
	}

	return len(list), nil
}

func (i *LeetCodeIngester) worker(
	ctx context.Context,
	jobs <-chan leetcode.QuestionSummary,
	results chan<- models.Problem,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for summary := range jobs {
		problemModel, err := i.getProblem(summary)
		if err != nil {
			if errors.Is(err, ErrPaidProblem) {
				log.Printf("skipping paid problem: %s", summary.TitleSlug)
				continue
			}

			log.Printf("failed to fetch %s: %v", summary.TitleSlug, err)
			continue
		}

		select {
		case results <- problemModel:
		case <-ctx.Done():
			return
		}
	}
}

func (i *LeetCodeIngester) getProblem(question leetcode.QuestionSummary) (models.Problem, error) {
	if question.PaidOnly {
		return models.Problem{}, ErrPaidProblem
	}

	questionData, err := i.client.FetchQuestionDetails(question.TitleSlug)

	if err != nil {
		return models.Problem{}, fmt.Errorf("failed to fetch question details: %w", err)
	}

	cleanedProblem, err := text.HTMLToText(questionData.Content)
	if err != nil {
		return models.Problem{}, fmt.Errorf("error cleaning problem: %w", err)
	}

	time.Sleep(i.requestDelay)

	problemModel := leetcode.ToProblem(questionData, cleanedProblem)
	return problemModel, nil
}
