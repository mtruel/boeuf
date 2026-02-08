package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
		log.Fatal("SESSION_KEY must be set and exactly 32 bytes")
	}
	if encryptionKey == "" || len(encryptionKey) != 32 {
		log.Fatal("ENCRYPTION_KEY must be set and exactly 32 bytes")
	}
	if spotifyClientID == "" {
		log.Fatal("SPOTIFY_CLIENT_ID must be set")
	}

	// Initialize database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	// MVP: Using GORM AutoMigrate for simplicity.
	// For a production-grade release later, we should switch to `goose` or `golang-migrate`
	// to handle complex schema changes that GORM cannot automate.
	if err := db.AutoMigrate(&models.SpotifyToken{}, &models.Session{}, &models.SessionInvite{}, &models.SessionParticipant{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
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

	// Initialize handlers
	authHandler := handlers.NewSpotifyAuthHandler(sessionStore)
	authHandler.SetTokenService(tokenService)
	authStatusHandler := handlers.NewAuthStatusHandler(sessionStore, tokenRepo)
	sessionHandler := handlers.NewSessionHandlerWithDuration(sessionStore, db, publicURL, sessionDuration)
	sessionHandler.SetMaxActiveSessionsPerUser(maxActiveSessionsPerUser)
	wsHandler := handlers.NewWebSocketHandler(hub, sessionStore, db)

	// Configure hub to broadcast PARTICIPANT_LEFT on disconnection
	hub.SetOnClientDisconnect(func(sessionID, userID string) {
		wsHandler.BroadcastParticipantLeft(sessionID, userID)
	})

	// Initialize session cleanup
	sessionCleaner := handlers.NewSessionCleaner(db)

	// Start periodic cleanup in background (every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run initial cleanup on startup
		if err := sessionCleaner.RunPeriodicCleanup(); err != nil {
			log.Printf("WARNING: Initial session cleanup failed: %v", err)
		}

		// Run periodic cleanup
		for range ticker.C {
			if err := sessionCleaner.RunPeriodicCleanup(); err != nil {
				log.Printf("WARNING: Periodic session cleanup failed: %v", err)
			}
		}
	}()

	// Setup routes with gorilla/mux for path variable support
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/api/health", healthHandler)
	router.HandleFunc("/auth/spotify/start", authHandler.Start)
	router.HandleFunc("/auth/spotify/callback", authHandler.Callback)
	router.HandleFunc("/api/auth/logout", authHandler.Logout).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/status", authStatusHandler.Status)
	router.HandleFunc("/api/sessions", sessionHandler.Create).Methods(http.MethodPost)
	router.HandleFunc("/api/sessions/join", sessionHandler.Join).Methods(http.MethodPost)

	// WebSocket route (requires authentication)
	router.HandleFunc("/ws/{sessionId}", wsHandler.HandleConnection)

	// Protected routes (require participant access control)
	accessControl := handlers.NewAccessControlMiddleware(sessionStore, db)
	// Example protected route - will be used in future stories for session data access
	router.Handle("/api/sessions/{sessionId}", accessControl.RequireParticipant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a placeholder - real implementation will come in future stories
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"protected_endpoint_accessed"}`))
	}))).Methods(http.MethodGet)

	// Enable CORS for development
	handler := corsMiddleware(router)

	// Store spotify client in context for future use
	_ = spotifyClient // Will be used in future stories

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Backend server starting on %s", addr)
	log.Printf("Database: %s", dbPath)
	log.Printf("Environment: %s", getEnv("ENV", "development"))

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
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
