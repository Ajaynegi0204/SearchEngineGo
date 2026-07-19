package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"problem-search/internal/models"
)

func (s *PostgresStore) CreateUser(
	ctx context.Context,
	email string,
	passwordHash string,
) (models.User, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash
	`

	var newUser models.User
	if err := s.pool.QueryRow(ctx, query, email, passwordHash).Scan(
		&newUser.ID,
		&newUser.Email,
		&newUser.PasswordHash,
	); err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	return newUser, nil
}

func (s *PostgresStore) GetUserByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	const query = `
		SELECT id, email, password_hash
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	return s.getUser(ctx, query, email)
}

func (s *PostgresStore) GetUserByID(
	ctx context.Context,
	id uuid.UUID,
) (models.User, error) {
	const query = `
		SELECT id, email, password_hash
		FROM users
		WHERE id = $1
	`

	return s.getUser(ctx, query, id)
}

func (s *PostgresStore) getUser(
	ctx context.Context,
	query string,
	argument any,
) (models.User, error) {
	var foundUser models.User
	if err := s.pool.QueryRow(ctx, query, argument).Scan(
		&foundUser.ID,
		&foundUser.Email,
		&foundUser.PasswordHash,
	); err != nil {
		return models.User{}, fmt.Errorf("get user: %w", err)
	}

	return foundUser, nil
}
