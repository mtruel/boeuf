package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("handler returned unexpected status: got %v want ok", response.Status)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want application/json", contentType)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_ENV_KEY", "")
	if value := getEnv("TEST_ENV_KEY", "default"); value != "default" {
		t.Errorf("expected default value, got %s", value)
	}

	t.Setenv("TEST_ENV_KEY", "override")
	if value := getEnv("TEST_ENV_KEY", "default"); value != "override" {
		t.Errorf("expected override value, got %s", value)
	}
}

func TestRunConfigErrors(t *testing.T) {
	validSessionKey := "12345678901234567890123456789012"
	validEncryptionKey := "abcdefghijklmnopqrstuvwxyz123456"

	tests := []struct {
		name          string
		sessionKey    string
		encryptionKey string
		clientID      string
		expectError   string
	}{
		{
			name:          "missing session key",
			sessionKey:    "",
			encryptionKey: validEncryptionKey,
			clientID:      "client-id",
			expectError:   "SESSION_KEY",
		},
		{
			name:          "invalid session key length",
			sessionKey:    "short",
			encryptionKey: validEncryptionKey,
			clientID:      "client-id",
			expectError:   "SESSION_KEY",
		},
		{
			name:          "missing encryption key",
			sessionKey:    validSessionKey,
			encryptionKey: "",
			clientID:      "client-id",
			expectError:   "ENCRYPTION_KEY",
		},
		{
			name:          "invalid encryption key length",
			sessionKey:    validSessionKey,
			encryptionKey: "short",
			clientID:      "client-id",
			expectError:   "ENCRYPTION_KEY",
		},
		{
			name:          "missing spotify client id",
			sessionKey:    validSessionKey,
			encryptionKey: validEncryptionKey,
			clientID:      "",
			expectError:   "SPOTIFY_CLIENT_ID",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SESSION_KEY", test.sessionKey)
			t.Setenv("ENCRYPTION_KEY", test.encryptionKey)
			t.Setenv("SPOTIFY_CLIENT_ID", test.clientID)
			t.Setenv("DATABASE_PATH", ":memory:")

			err := run()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), test.expectError) {
				t.Fatalf("expected error to contain %q, got %v", test.expectError, err)
			}
		})
	}
}

func TestRunDatabaseError(t *testing.T) {
	validSessionKey := "12345678901234567890123456789012"
	validEncryptionKey := "abcdefghijklmnopqrstuvwxyz123456"

	t.Setenv("SESSION_KEY", validSessionKey)
	t.Setenv("ENCRYPTION_KEY", validEncryptionKey)
	t.Setenv("SPOTIFY_CLIENT_ID", "client-id")
	t.Setenv("DATABASE_PATH", "/nonexistent/boeuf.db")

	err := run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to connect to database") {
		t.Fatalf("expected database error, got %v", err)
	}
}
