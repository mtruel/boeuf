package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/mathias/boeuf/internal/auth"
	"github.com/mathias/boeuf/internal/session"
	"github.com/mathias/boeuf/internal/spotify"
)

type SpotifyAuthHandler struct {
	store        *sessions.CookieStore
	tokenService *spotify.TokenService
}

func NewSpotifyAuthHandler(store *sessions.CookieStore) *SpotifyAuthHandler {
	return &SpotifyAuthHandler{
		store:        store,
		tokenService: nil, // Will be set later when needed
	}
}

func (h *SpotifyAuthHandler) SetTokenService(service *spotify.TokenService) {
	h.tokenService = service
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

	// Exchange code for tokens
	if h.tokenService != nil {
		result, err := h.tokenService.ExchangeCodeForTokens(r.Context(), code, codeVerifier, clientID, redirectURI)
		if err != nil {
			log.Printf("ERROR: Token exchange failed: %v", err)
			http.Error(w, "Token exchange failed", http.StatusInternalServerError)
			return
		}

		// Store spotify_user_id in session
		sess.Values["spotify_user_id"] = result.SpotifyUserID
	}

	// Clean up session PKCE data
	delete(sess.Values, "state")
	delete(sess.Values, "code_verifier")
	sess.Save(r, w)

	// Redirect to frontend
	http.Redirect(w, r, "/", http.StatusFound)
}
