package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"problem-search/internal/models"
)

func (s *PostgresStore) CreateRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) error {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`

	if _, err := s.pool.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetRefreshToken(
	ctx context.Context,
	tokenHash string,
) (models.RefreshToken, error) {
	const query = `
		SELECT id, user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token models.RefreshToken
	if err := s.pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.ExpiresAt,
		&token.RevokedAt,
	); err != nil {
		return models.RefreshToken{}, fmt.Errorf("get refresh token: %w", err)
	}

	return token, nil
}

func (s *PostgresStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	if _, err := s.pool.Exec(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}
