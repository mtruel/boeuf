package handlers

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mathias/boeuf/internal/auth"
	"github.com/mathias/boeuf/internal/session"
	"github.com/mathias/boeuf/internal/spotify"
)

type SpotifyAuthHandler struct {
	store        session.Store
	tokenService *spotify.TokenService
}

func NewSpotifyAuthHandler(store session.Store, tokenService *spotify.TokenService) *SpotifyAuthHandler {
	return &SpotifyAuthHandler{
		store:        store,
		tokenService: tokenService,
	}
}

func isAllowedReturnTo(returnTo string) bool {
	parsed, err := url.Parse(returnTo)
	if err != nil {
		return false
	}

	if parsed.Scheme != "" || parsed.Host != "" {
		if os.Getenv("RUNTIME_ENV") != "production" {
			host := parsed.Hostname()
			if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
				return true
			}
		}

		publicURL := os.Getenv("PUBLIC_URL")
		if publicURL == "" {
			return false
		}

		parsedPublic, err := url.Parse(publicURL)
		if err != nil {
			log.Printf("ERROR: Invalid PUBLIC_URL: %v", err)
			return false
		}

		return parsed.Hostname() == parsedPublic.Hostname()
	}

	return strings.HasPrefix(parsed.Path, "/")
}

// Start initiates the OAuth flow by redirecting to Spotify
func (h *SpotifyAuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	// Get config from environment
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")

	if clientID == "" || redirectURI == "" {
		log.Printf("ERROR: Missing Spotify OAuth config")
		http.Error(w, "OAuth configuration error", http.StatusInternalServerError)
		return
	}

	// Generate PKCE parameters
	codeVerifier, err := auth.GenerateCodeVerifier()
	if err != nil {
		log.Printf("ERROR: Failed to generate code verifier: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	codeChallenge := auth.GenerateCodeChallenge(codeVerifier)

	state, err := auth.GenerateState()
	if err != nil {
		log.Printf("ERROR: Failed to generate state: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state and code_verifier in session (server-side)
	sess, err := h.store.Get(r, session.SessionName)
	if err != nil {
		log.Printf("ERROR: Failed to get session: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	sess.Values["state"] = state
	sess.Values["code_verifier"] = codeVerifier

	// Store return_to URL if provided (for post-OAuth redirect)
	if returnTo := r.URL.Query().Get("return_to"); returnTo != "" {
		if isAllowedReturnTo(returnTo) {
			sess.Values["return_to"] = returnTo
		} else {
			log.Printf("WARN: Rejected return_to parameter: %s", returnTo)
		}
	}

	err = sess.Save(r, w)
	if err != nil {
		log.Printf("ERROR: Failed to save session: %v", err)
		http.Error(w, "Session save error", http.StatusInternalServerError)
		return
	}

	// Build authorization URL
	authURL := auth.BuildAuthorizationURL(auth.AuthURLParams{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		CodeChallenge: codeChallenge,
		State:         state,
		Scopes:        auth.RequiredScopes(),
	})

	// Redirect to Spotify
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles the OAuth callback from Spotify
func (h *SpotifyAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	// Get session
	sess, err := h.store.Get(r, session.SessionName)
	if err != nil {
		log.Printf("ERROR: Failed to get session in callback: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	// Validate state (CSRF protection)
	expectedState, ok := sess.Values["state"].(string)
	if !ok || expectedState == "" {
		log.Printf("ERROR: No state in session")
		http.Error(w, "Invalid session state", http.StatusBadRequest)
		return
	}

	receivedState := r.URL.Query().Get("state")
	if receivedState != expectedState {
		log.Printf("ERROR: State mismatch - expected %s, got %s", expectedState, receivedState)
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		// Check for error from Spotify
		spotifyError := r.URL.Query().Get("error")
		log.Printf("ERROR: OAuth failed - %s", spotifyError)
		http.Error(w, "OAuth authorization failed", http.StatusBadRequest)
		return
	}

	// Get code_verifier from session
	codeVerifier, ok := sess.Values["code_verifier"].(string)
	if !ok || codeVerifier == "" {
		log.Printf("ERROR: No code_verifier in session")
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	// Get config
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")

	if h.tokenService == nil {
		log.Printf("ERROR: Token service not initialized")
		http.Error(w, "Token service not initialized", http.StatusInternalServerError)
		return
	}

	// Exchange code for tokens
	result, err := h.tokenService.ExchangeCodeForTokens(r.Context(), code, codeVerifier, clientID, redirectURI)
	if err != nil {
		log.Printf("ERROR: Token exchange failed: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Store spotify_user_id in session
	sess.Values["spotify_user_id"] = result.SpotifyUserID

	// Clean up session PKCE data
	delete(sess.Values, "state")
	delete(sess.Values, "code_verifier")

	// Get return_to URL from session (if set during Start)
	returnTo, _ := sess.Values["return_to"].(string)
	delete(sess.Values, "return_to") // Clean up after reading
	if returnTo != "" && !isAllowedReturnTo(returnTo) {
		log.Printf("WARN: Rejected return_to from session: %s", returnTo)
		returnTo = ""
	}

	if err := sess.Save(r, w); err != nil {
		log.Printf("ERROR: Failed to save session in callback: %v", err)
		http.Error(w, "Session save error", http.StatusInternalServerError)
		return
	}

	// Redirect to frontend - use return_to if present, otherwise default to /
	redirectURL := "/"
	if returnTo != "" {
		redirectURL = returnTo
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// Logout clears the Spotify authentication from the session
func (h *SpotifyAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session
	sess, err := h.store.Get(r, session.SessionName)
	if err != nil {
		log.Printf("ERROR: Failed to get session in logout: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	// Clear spotify authentication data
	delete(sess.Values, "spotify_user_id")

	// Save updated session
	err = sess.Save(r, w)
	if err != nil {
		log.Printf("ERROR: Failed to save session in logout: %v", err)
		http.Error(w, "Session save error", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: User logged out successfully")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"success":true}`)); err != nil {
		log.Printf("ERROR: Failed to write logout response: %v", err)
	}
}
