package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mathias/boeuf/internal/handlers"
	"github.com/mathias/boeuf/internal/models"
	"github.com/mathias/boeuf/internal/realtime"
	"github.com/mathias/boeuf/internal/repository"
	"github.com/mathias/boeuf/internal/session"
	"github.com/mathias/boeuf/internal/spotify"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	port := getEnv("PORT", "8080")
	sessionKey := getEnv("SESSION_KEY", "")
	encryptionKey := getEnv("ENCRYPTION_KEY", "")
	dbPath := getEnv("DATABASE_PATH", "./data/boeuf.db")
	publicURL := getEnv("PUBLIC_URL", "http://localhost:3000")
	spotifyClientID := getEnv("SPOTIFY_CLIENT_ID", "")
	isProduction := getEnv("ENV", "development") == "production"

	// Parse session duration from environment (in hours, default 24)
	sessionDurationHours, err := strconv.Atoi(getEnv("SESSION_DURATION_HOURS", "24"))
	if err != nil || sessionDurationHours <= 0 {
		log.Printf("WARNING: Invalid SESSION_DURATION_HOURS, using default 24 hours")
		sessionDurationHours = 24
	}
	sessionDuration := time.Duration(sessionDurationHours) * time.Hour

	// Parse max active sessions per user (0 = unlimited, default 10)
	maxActiveSessionsPerUser, err := strconv.Atoi(getEnv("MAX_ACTIVE_SESSIONS_PER_USER", "10"))
	if err != nil || maxActiveSessionsPerUser < 0 {
		log.Printf("WARNING: Invalid MAX_ACTIVE_SESSIONS_PER_USER, using default 10")
		maxActiveSessionsPerUser = 10
	}

	// Validate required config
	if sessionKey == "" || len(sessionKey) != 32 {
		return fmt.Errorf("SESSION_KEY must be set and exactly 32 bytes")
	}
	if encryptionKey == "" || len(encryptionKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be set and exactly 32 bytes")
	}
	if spotifyClientID == "" {
		return fmt.Errorf("SPOTIFY_CLIENT_ID must be set")
	}

	// Initialize database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	// MVP: Using GORM AutoMigrate for simplicity.
	// For a production-grade release later, we should switch to `goose` or `golang-migrate`
	// to handle complex schema changes that GORM cannot automate.
	if err := db.AutoMigrate(&models.SpotifyToken{}, &models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{}, &models.Event{}); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize repositories and services
	tokenRepo := repository.NewSpotifyTokenRepository(db)
	tokenService := spotify.NewTokenService(tokenRepo, encryptionKey)
	spotifyClient := spotify.NewClient(tokenRepo, encryptionKey, spotifyClientID)

	// Ensure we satisfy the abstraction
	var _ spotify.Provider = spotifyClient

	// Initialize session store
	sessionStore := session.NewStore(sessionKey, isProduction)

	// Initialize WebSocket hub and start it
	hub := realtime.NewHub()
	go hub.Run()

	// Initialize PlayerPoller for continuous player state sync
	playerPoller := session.NewPlayerPoller(db, spotifyClient, hub, 5*time.Second)

	// Initialize handlers
	authHandler := handlers.NewSpotifyAuthHandler(sessionStore, tokenService)
	authStatusHandler := handlers.NewAuthStatusHandler(sessionStore, tokenRepo)
	sessionHandler := handlers.NewSessionHandlerWithDuration(sessionStore, db, publicURL, sessionDuration)
	sessionHandler.SetMaxActiveSessionsPerUser(maxActiveSessionsPerUser)
	sessionHandler.SetSpotifyClient(spotifyClient)
	sessionHandler.SetRealtimeHub(hub)
	wsHandler := handlers.NewWebSocketHandler(hub, sessionStore, db)
	playerHandler := handlers.NewPlayerHandler(sessionStore, db, spotifyClient, hub)

	// Configure hub to start/stop polling based on participant presence
	hub.SetOnClientConnect(func(sessionID, userID string) {
		if !playerPoller.IsPolling(sessionID) {
			playerPoller.StartSessionPolling(sessionID)
		}
	})

	// Configure hub to broadcast PARTICIPANT_LEFT on disconnection + stop polling
	hub.SetOnClientDisconnect(func(sessionID, userID string) {
		wsHandler.BroadcastParticipantLeft(sessionID, userID)
		clients := hub.GetSessionClients(sessionID)
		if len(clients) == 0 {
			playerPoller.StopSessionPolling(sessionID)
		}
	})

	// Initialize session cleanup
	sessionCleaner := handlers.NewSessionCleaner(db)
	cleanupService := handlers.NewCleanupService(sessionCleaner, time.Hour)
	cleanupService.Start()

	// Setup routes with gorilla/mux for path variable support
	router := mux.NewRouter()

	// Public routes
	registerRoutes(router, []routeDefinition{
		{method: http.MethodGet, path: "/api/health", handler: healthHandler},
		{method: http.MethodGet, path: "/auth/spotify/start", handler: authHandler.Start},
		{method: http.MethodGet, path: "/auth/spotify/callback", handler: authHandler.Callback},
		{method: http.MethodPost, path: "/api/auth/logout", handler: authHandler.Logout},
		{method: http.MethodGet, path: "/api/auth/status", handler: authStatusHandler.Status},
		{method: http.MethodPost, path: "/api/sessions", handler: sessionHandler.Create},
		{method: http.MethodPost, path: "/api/sessions/join", handler: sessionHandler.Join},
		{method: http.MethodGet, path: "/ws/{sessionId}", handler: wsHandler.HandleConnection},
	})

	// Protected routes (require participant access control)
	accessControl := handlers.NewAccessControlMiddleware(sessionStore, db)
	registerParticipantRoutes(router, accessControl, []routeDefinition{
		{method: http.MethodGet, path: "/api/sessions/{sessionId}", handler: sessionHandler.GetSession},
		{method: http.MethodGet, path: "/api/sessions/{sessionId}/me", handler: sessionHandler.GetParticipantMe},
		{method: http.MethodPost, path: "/api/sessions/{sessionId}/sync/start", handler: sessionHandler.StartSync},
		{method: http.MethodGet, path: "/api/sessions/{sessionId}/player/state", handler: playerHandler.GetPlayerState},
		{method: http.MethodPost, path: "/api/sessions/{sessionId}/player/pause", handler: playerHandler.PausePlayer},
		{method: http.MethodPost, path: "/api/sessions/{sessionId}/player/resume", handler: playerHandler.ResumePlayer},
		{method: http.MethodPost, path: "/api/sessions/{sessionId}/player/next", handler: playerHandler.NextTrack},
		{method: http.MethodPost, path: "/api/sessions/{sessionId}/player/seek", handler: playerHandler.SeekPlayer},
	})

	// Enable CORS for development
	handler := corsMiddleware(router)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Backend server starting on %s", addr)
	log.Printf("Database: %s", dbPath)
	log.Printf("Environment: %s", getEnv("ENV", "development"))

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			playerPoller.StopAll()
			cleanupService.Stop(cleanupCtx)
			return fmt.Errorf("server error: %w", err)
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		playerPoller.StopAll()
		cleanupService.Stop(cleanupCtx)
		return nil
	case <-ctx.Done():
		log.Printf("Shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		playerPoller.StopAll()
		cleanupService.Stop(shutdownCtx)
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	playerPoller.StopAll()
	cleanupService.Stop(shutdownCtx)
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := os.Getenv("ENV")
		if env == "production" {
			// In production, Caddy handles CORS - don't set wildcard
			next.ServeHTTP(w, r)
			return
		}

		// Development mode only - specific origin required when using credentials
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "http://localhost:3000" // Default dev origin
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type routeDefinition struct {
	method  string
	path    string
	handler http.HandlerFunc
}

func registerRoutes(router *mux.Router, routes []routeDefinition) {
	for _, route := range routes {
		h := router.HandleFunc(route.path, route.handler)
		if route.method != "" {
			h.Methods(route.method)
		}
	}
}

func registerParticipantRoutes(router *mux.Router, accessControl *handlers.AccessControlMiddleware, routes []routeDefinition) {
	for _, route := range routes {
		h := router.Handle(route.path, accessControl.RequireParticipant(http.HandlerFunc(route.handler)))
		if route.method != "" {
			h.Methods(route.method)
		}
	}
}
