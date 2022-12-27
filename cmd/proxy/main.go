package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	githubClientSecret string
	githubClientID     string
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Debug().Msg("Starting reverse proxy")

	// Load Github client secret from environment variables
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if githubClientSecret == "" {
		log.Error().Msg("GITHUB_CLIENT_SECRET is not set")
		os.Exit(1)
	}

	// Load Github client ID from environment variables
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	if githubClientID == "" {
		log.Error().Msg("GITHUB_CLIENT_ID is not set")
		os.Exit(1)
	}

	// Create router
	r := chi.NewRouter()
	r.Use(middleware.Recoverer, middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://www.youryearincode.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Get("/", testHandler)
	r.Post("/oauth", oauthHandler)

	// Start router
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Error().Err(err).Caller().Msg("error while listening and serving")
		os.Exit(1)
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello world!"))
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the token value from the query parameters
	tokenQueryParam := r.URL.Query().Get("code")
	if tokenQueryParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create access token request
	url := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s", githubClientID, githubClientSecret, tokenQueryParam)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("Accept", "application/json")
	if err != nil {
		log.Error().Err(err).Caller().Msg("error while creating Github request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error while exchanging token for Github access token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().Str("status", resp.Status).Caller().Msg("error while exchanging token for Github access token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Read Github response into buffer
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		log.Error().Err(err).Caller().Msg("error while reading Github response into buffer")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write response
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Error().Err(err).Caller().Msg("error while writing response")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
