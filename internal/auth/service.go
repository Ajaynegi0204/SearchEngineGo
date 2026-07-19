package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"problem-search/internal/models"
	"problem-search/internal/storage"
)

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 30 * 24 * time.Hour
)

var (
	ErrEmailAlreadyRegistered = errors.New("email is already registered")
	ErrInvalidInput           = errors.New("invalid authentication input")
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Service struct {
	store     *storage.PostgresStore
	jwtSecret []byte
}

func NewService(store *storage.PostgresStore, jwtSecret string) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("postgres store is required")
	}
	if strings.TrimSpace(jwtSecret) == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &Service{store: store, jwtSecret: []byte(jwtSecret)}, nil
}

func (s *Service) Signup(ctx context.Context, email, password string) (models.User, error) {
	email, err := validateCredentials(email, password)
	if err != nil {
		return models.User{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("hash password: %w", err)
	}

	newUser, err := s.store.CreateUser(ctx, email, string(passwordHash))
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" &&
			postgresError.ConstraintName == "users_email_lower_unique" {
			return models.User{}, ErrEmailAlreadyRegistered
		}
		return models.User{}, err
	}

	return newUser, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (models.User, Tokens, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return models.User{}, Tokens{}, fmt.Errorf("%w: email and password are required", ErrInvalidInput)
	}

	foundUser, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, Tokens{}, ErrInvalidCredentials
		}
		return models.User{}, Tokens{}, fmt.Errorf("get user for login: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(password)); err != nil {
		return models.User{}, Tokens{}, ErrInvalidCredentials
	}

	tokens, err := s.createTokens(ctx, foundUser.ID)
	if err != nil {
		return models.User{}, Tokens{}, err
	}
	return foundUser, tokens, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return Tokens{}, fmt.Errorf("%w: refresh token is required", ErrInvalidInput)
	}

	tokenRecord, err := s.store.GetRefreshToken(ctx, hashToken(refreshToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Tokens{}, ErrInvalidRefreshToken
		}
		return Tokens{}, fmt.Errorf("get refresh token: %w", err)
	}
	if tokenRecord.RevokedAt != nil || !tokenRecord.ExpiresAt.After(time.Now()) {
		return Tokens{}, ErrInvalidRefreshToken
	}

	accessToken, err := s.createAccessToken(tokenRecord.UserID)
	if err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTokenDuration.Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	return s.store.RevokeRefreshToken(ctx, hashToken(refreshToken))
}

func (s *Service) ValidateAccessToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid access token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid access token claims")
	}

	userIDString, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("access token subject is missing")
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid access token subject")
	}
	return userID, nil
}

func (s *Service) createTokens(ctx context.Context, userID uuid.UUID) (Tokens, error) {
	accessToken, err := s.createAccessToken(userID)
	if err != nil {
		return Tokens{}, err
	}

	refreshToken, err := createRandomToken()
	if err != nil {
		return Tokens{}, fmt.Errorf("create refresh token: %w", err)
	}

	expiresAt := time.Now().Add(refreshTokenDuration)
	if err := s.store.CreateRefreshToken(ctx, userID, hashToken(refreshToken), expiresAt); err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTokenDuration.Seconds()),
	}, nil
}

func (s *Service) createAccessToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenDuration)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signedToken, nil
}

func validateCredentials(email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password are required")
	}
	if len(password) < 8 {
		return "", fmt.Errorf("password must contain at least 8 characters")
	}
	if len(password) > 72 {
		return "", fmt.Errorf("password must not exceed 72 characters")
	}
	return email, nil
}

func createRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
