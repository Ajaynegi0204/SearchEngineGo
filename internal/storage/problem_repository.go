package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"problem-search/internal/models"
)

func (s *PostgresStore) UpsertProblems(ctx context.Context, problems []models.Problem) error {
	if len(problems) == 0 {
		return nil
	}

	const query = `
		INSERT INTO problems (
			platform,
			external_id,
			slug,
			title,
			url,
			difficulty,
			tags,
			statement_text
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (platform, slug)
		DO UPDATE SET
			external_id = EXCLUDED.external_id,
			title = EXCLUDED.title,
			url = EXCLUDED.url,
			difficulty = EXCLUDED.difficulty,
			tags = EXCLUDED.tags,
			statement_text = EXCLUDED.statement_text,
			updated_at = NOW()
	`

	batch := &pgx.Batch{}
	for _, p := range problems {
		batch.Queue(
			query,
			p.Platform,
			p.ExternalID,
			p.Slug,
			p.Title,
			p.URL,
			p.Difficulty,
			p.Tags,
			p.StatementText,
		)
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range problems {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert problems batch: %w", err)
		}
	}

	return nil
}

func (s *PostgresStore) ListProblems(ctx context.Context, lastID int64, limit int) ([]models.Problem, int64, error) {
	if limit <= 0 || limit > 100 || lastID < 0 {
		return nil, 0, fmt.Errorf("invalid arguments")
	}

	const query = `
		SELECT id, platform, external_id, slug, title, url, difficulty, tags, statement_text
		FROM problems
		WHERE id > $1
		ORDER BY id
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, lastID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list problem query: %w", err)
	}
	defer rows.Close()

	var allProblems []models.Problem
	nextID := lastID
	for rows.Next() {
		var p models.Problem
		var rowID int64
		err := rows.Scan(
			&rowID,
			&p.Platform,
			&p.ExternalID,
			&p.Slug,
			&p.Title,
			&p.URL,
			&p.Difficulty,
			&p.Tags,
			&p.StatementText,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("scan problem row: %w", err)
		}

		p.ID = rowID
		nextID = rowID
		allProblems = append(allProblems, p)
	}

	if rows.Err() != nil {
		return nil, 0, fmt.Errorf("iterate problem rows: %w", rows.Err())
	}
	return allProblems, nextID, nil
}

func (s *PostgresStore) ListAllProblems(ctx context.Context, pageSize int) ([]models.Problem, error) {
	if pageSize <= 0 || pageSize > 100 {
		return nil, fmt.Errorf("page size must be between 1 and 100")
	}

	var allProblems []models.Problem
	var lastID int64

	for {
		page, nextID, err := s.ListProblems(ctx, lastID, pageSize)
		if err != nil {
			return nil, fmt.Errorf("list problems after id %d: %w", lastID, err)
		}
		if len(page) == 0 {
			break
		}

		allProblems = append(allProblems, page...)
		lastID = nextID
		if len(page) < pageSize {
			break
		}
	}

	return allProblems, nil
}

func (s *PostgresStore) GetProblemsByIDs(ctx context.Context, ids []int64) (map[int64]models.Problem, error) {
	if len(ids) == 0 {
		return map[int64]models.Problem{}, nil
	}

	const query = `
		SELECT id, platform, external_id, slug, title, url, difficulty, tags, statement_text
		FROM problems
		WHERE id = ANY($1)
	`

	rows, err := s.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("get problems by IDs: %w", err)
	}
	defer rows.Close()

	problems := make(map[int64]models.Problem, len(ids))
	for rows.Next() {
		var p models.Problem
		var rowID int64

		if err := rows.Scan(
			&rowID,
			&p.Platform,
			&p.ExternalID,
			&p.Slug,
			&p.Title,
			&p.URL,
			&p.Difficulty,
			&p.Tags,
			&p.StatementText,
		); err != nil {
			return nil, fmt.Errorf("scan problem by ID: %w", err)
		}

		p.ID = rowID
		problems[rowID] = p
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate problems by ID: %w", err)
	}

	return problems, nil
}
