package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAccessControlTestDB(t *testing.T) (*gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.AutoMigrate(&models.Session{}, &models.SessionParticipant{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

func TestRequireParticipant_Success(t *testing.T) {
	db, cleanup := setupAccessControlTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))

	// Create test session and participant
	sessionID := "sess_test_access"
	userID := "user_test_123"

	testSession := models.Session{
		ID:     sessionID,
		Active: true,
	}
	db.Create(&testSession)

	participant := models.SessionParticipant{
		SessionID: sessionID,
		UserID:    userID,
		Role:      "participant",
	}
	db.Create(&participant)

	// Create a test handler that will be protected
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Access granted"))
	})

	// Wrap it with the middleware using mux router
	middleware := NewAccessControlMiddleware(store, db)
	router := mux.NewRouter()
	router.Handle("/sessions/{sessionId}/data", middleware.RequireParticipant(protectedHandler))

	// Create authenticated request
	req := httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID+"/data", nil)
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = userID
	session.Values["authenticated"] = true

	w := httptest.NewRecorder()
	session.Save(req, w)
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w = httptest.NewRecorder()

	// Call the router (which will extract the sessionId variable)
	router.ServeHTTP(w, req)

	// Verify access was granted
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if w.Body.String() != "Access granted" {
		t.Errorf("Expected 'Access granted', got %s", w.Body.String())
	}
}

func TestRequireParticipant_NotParticipant(t *testing.T) {
	db, cleanup := setupAccessControlTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))

	// Create test session without adding user as participant
	sessionID := "sess_forbidden"
	userID := "user_not_participant"

	testSession := models.Session{
		ID:     sessionID,
		Active: true,
	}
	db.Create(&testSession)

	// Create a test handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Access granted"))
	})

	// Wrap it with the middleware using mux router
	middleware := NewAccessControlMiddleware(store, db)
	router := mux.NewRouter()
	router.Handle("/sessions/{sessionId}/data", middleware.RequireParticipant(protectedHandler))

	// Create authenticated request (but user is not a participant)
	req := httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID+"/data", nil)
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = userID
	session.Values["authenticated"] = true

	w := httptest.NewRecorder()
	session.Save(req, w)
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w = httptest.NewRecorder()

	// Call the router
	router.ServeHTTP(w, req)

	// Verify access was denied
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	// Verify error response format
	body := w.Body.String()
	if !contains(body, "FORBIDDEN") {
		t.Errorf("Expected FORBIDDEN error code, got: %s", body)
	}
}

func TestRequireParticipant_Unauthenticated(t *testing.T) {
	db, cleanup := setupAccessControlTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))

	sessionID := "sess_unauth"

	// Create a test handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Access granted"))
	})

	// Wrap it with the middleware using mux router
	middleware := NewAccessControlMiddleware(store, db)
	router := mux.NewRouter()
	router.Handle("/sessions/{sessionId}/data", middleware.RequireParticipant(protectedHandler))

	// Create unauthenticated request
	req := httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID+"/data", nil)
	w := httptest.NewRecorder()

	// Call the router
	router.ServeHTTP(w, req)

	// Verify access was denied
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	// Verify error response format
	body := w.Body.String()
	if !contains(body, "UNAUTHENTICATED") {
		t.Errorf("Expected UNAUTHENTICATED error code, got: %s", body)
	}
}

func TestRequireParticipant_SessionNotFound(t *testing.T) {
	db, cleanup := setupAccessControlTestDB(t)
	defer cleanup()

	store := sessions.NewCookieStore([]byte("test-key-32-bytes-long-for-hmac"))

	// Use non-existent session ID
	sessionID := "sess_nonexistent"
	userID := "user_test"

	// Create a test handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Access granted"))
	})

	// Wrap it with the middleware using mux router
	middleware := NewAccessControlMiddleware(store, db)
	router := mux.NewRouter()
	router.Handle("/sessions/{sessionId}/data", middleware.RequireParticipant(protectedHandler))

	// Create authenticated request
	req := httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID+"/data", nil)
	session, _ := store.Get(req, "boeuf-session")
	session.Values["spotify_user_id"] = userID
	session.Values["authenticated"] = true

	w := httptest.NewRecorder()
	session.Save(req, w)
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w = httptest.NewRecorder()

	// Call the router
	router.ServeHTTP(w, req)

	// Verify access was denied with FORBIDDEN (not SESSION_NOT_FOUND to avoid info leak)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// Helper function to check if string contains substring (wrapper for strings.Contains)
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
