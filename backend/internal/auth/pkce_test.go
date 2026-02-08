package auth_test

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mathias/boeuf/internal/auth"
)

func TestGenerateCodeVerifier(t *testing.T) {
	// ACT
	verifier, err := auth.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("Failed to generate code verifier: %v", err)
	}

	// ASSERT
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("Code verifier length must be between 43-128, got %d", len(verifier))
	}

	// Should only contain URL-safe characters
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	for _, char := range verifier {
		if !strings.ContainsRune(validChars, char) {
			t.Errorf("Invalid character in code verifier: %c", char)
		}
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	// ARRANGE
	verifier := "test_code_verifier_1234567890abcdefghijklmnop"

	// ACT
	challenge := auth.GenerateCodeChallenge(verifier)

	// ASSERT
	// Manually compute expected challenge
	hash := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(hash[:])

	if challenge != expected {
		t.Errorf("Expected challenge %s, got %s", expected, challenge)
	}

	// Should be URL-safe base64 without padding
	if strings.Contains(challenge, "+") || strings.Contains(challenge, "/") || strings.Contains(challenge, "=") {
		t.Error("Challenge should be URL-safe base64 without padding")
	}
}

func TestGenerateState(t *testing.T) {
	// ACT
	state, err := auth.GenerateState()
	if err != nil {
		t.Fatalf("Failed to generate state: %v", err)
	}

	// ASSERT
	if len(state) < 16 {
		t.Errorf("State should be at least 16 chars, got %d", len(state))
	}

	// Generate multiple states to ensure randomness
	state2, err := auth.GenerateState()
	if err != nil {
		t.Fatalf("Failed to generate state2: %v", err)
	}
	if state == state2 {
		t.Error("Generated states should be unique")
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	// ARRANGE
	params := auth.AuthURLParams{
		ClientID:      "test_client_id",
		RedirectURI:   "http://localhost/callback",
		CodeChallenge: "test_challenge",
		State:         "test_state",
		Scopes:        []string{"user-read-playback-state", "user-modify-playback-state"},
	}

	// ACT
	url := auth.BuildAuthorizationURL(params)

	// ASSERT
	if !strings.HasPrefix(url, "https://accounts.spotify.com/authorize?") {
		t.Errorf("URL should start with Spotify auth endpoint, got: %s", url)
	}

	if !strings.Contains(url, "client_id=test_client_id") {
		t.Error("URL should contain client_id")
	}

	if !strings.Contains(url, "response_type=code") {
		t.Error("URL should contain response_type=code")
	}

	if !strings.Contains(url, "code_challenge_method=S256") {
		t.Error("URL should contain code_challenge_method=S256")
	}

	if !strings.Contains(url, "code_challenge=test_challenge") {
		t.Error("URL should contain code_challenge")
	}

	if !strings.Contains(url, "state=test_state") {
		t.Error("URL should contain state")
	}

	if !strings.Contains(url, "scope=user-read-playback-state+user-modify-playback-state") {
		t.Error("URL should contain scopes")
	}

	if !strings.Contains(url, "redirect_uri=") {
		t.Error("URL should contain redirect_uri")
	}
}

func TestRequiredScopes(t *testing.T) {
	// ACT
	scopes := auth.RequiredScopes()

	// ASSERT - As per story AC3, minimum required scopes
	requiredScopes := []string{
		"user-read-playback-state",
		"user-read-currently-playing",
		"user-modify-playback-state",
		"user-read-email",
	}

	if len(scopes) != len(requiredScopes) {
		t.Errorf("Expected %d scopes, got %d", len(requiredScopes), len(scopes))
	}

	for _, required := range requiredScopes {
		found := false
		for _, scope := range scopes {
			if scope == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing required scope: %s", required)
		}
	}
}
