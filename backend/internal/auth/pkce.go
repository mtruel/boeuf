package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

const (
	SpotifyAuthURL         = "https://accounts.spotify.com/authorize"
	pkceVerifierByteLength = 64
)

// GenerateCodeVerifier creates a cryptographically random code verifier
// as per RFC 7636 (43-128 characters, URL-safe)
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, pkceVerifierByteLength) // 64 bytes = 86 base64 chars (within 43-128 range)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateCodeChallenge creates the S256 code challenge from the verifier
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// GenerateState creates a random state parameter for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// AuthURLParams contains parameters for building Spotify authorization URL
type AuthURLParams struct {
	ClientID      string
	RedirectURI   string
	CodeChallenge string
	State         string
	Scopes        []string
}

// BuildAuthorizationURL constructs the Spotify OAuth authorization URL
func BuildAuthorizationURL(params AuthURLParams) string {
	v := url.Values{}
	v.Set("client_id", params.ClientID)
	v.Set("response_type", "code")
	v.Set("redirect_uri", params.RedirectURI)
	v.Set("code_challenge_method", "S256")
	v.Set("code_challenge", params.CodeChallenge)
	v.Set("state", params.State)
	v.Set("scope", strings.Join(params.Scopes, " "))

	return fmt.Sprintf("%s?%s", SpotifyAuthURL, v.Encode())
}

// RequiredScopes returns the minimum required Spotify scopes for boeuf
// As per story AC3: playback state, currently playing, modify playback, email
func RequiredScopes() []string {
	return []string{
		"user-read-playback-state",
		"user-read-currently-playing",
		"user-modify-playback-state",
		"user-read-email",
	}
}
