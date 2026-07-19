package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"problem-search/internal/api/handlers"
	"problem-search/internal/api/middleware"
	"problem-search/internal/auth"
	"problem-search/internal/search"
)

func NewRouter(authService *auth.Service, searchEngine *search.HybridEngine) http.Handler {
	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:5173",
			"http://localhost:4173",
			"https://codehunt-frontend.onrender.com",
		},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:         300,
	}))
	router.Get("/health", healthHandler)

	authHandler := handlers.NewAuthHandler(authService)
	queryHandler := handlers.NewQueryHandler(searchEngine)

	router.Post("/api/v1/auth/signup", authHandler.Signup)
	router.Post("/api/v1/auth/login", authHandler.Login)
	router.Post("/api/v1/auth/refresh", authHandler.Refresh)
	router.Post("/api/v1/auth/logout", authHandler.Logout)
	router.With(middleware.RequireAuth(authService)).Post("/query", queryHandler.Query)

	return router
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
