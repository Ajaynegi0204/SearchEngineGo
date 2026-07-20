package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	QdrantURL    string
	QdrantAPIKey string
	CohereAPIKey string
	JWTSecret    string
	Port         int
}

func Load() (Config, error) {
	_ = godotenv.Load()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	port := 8080
	if configuredPort := os.Getenv("PORT"); configuredPort != "" {
		parsedPort, err := strconv.Atoi(configuredPort)
		if err != nil || parsedPort < 1 || parsedPort > 65535 {
			return Config{}, fmt.Errorf("PORT must be a valid TCP port")
		}
		port = parsedPort
	}

	return Config{
		DatabaseURL:  databaseURL,
		QdrantURL:    os.Getenv("QDRANT_URL"),
		QdrantAPIKey: os.Getenv("QDRANT_API_KEY"),
		CohereAPIKey: os.Getenv("COHERE_API_KEY"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		Port:         port,
	}, nil
}
