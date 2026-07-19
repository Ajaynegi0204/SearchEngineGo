package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"problem-search/internal/auth"
)

type AuthHandler struct {
	authService *auth.Service
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	User   publicUser  `json:"user"`
	Tokens auth.Tokens `json:"tokens"`
}

type publicUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	newUser, err := h.authService.Signup(r.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, auth.ErrEmailAlreadyRegistered) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, auth.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("signup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}

	writeJSON(w, http.StatusCreated, publicUser{ID: newUser.ID.String(), Email: newUser.Email})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	loggedInUser, tokens, err := h.authService.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		log.Printf("login failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not log in")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		User:   publicUser{ID: loggedInUser.ID.String(), Email: loggedInUser.Email},
		Tokens: tokens,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		log.Printf("refresh token failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not refresh session")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authService.Logout(r.Context(), request.RefreshToken); err != nil {
		log.Printf("logout failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not log out")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
