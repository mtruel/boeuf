package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

var (
	spotifyOAuthConfig *oauth2.Config
	oauthStateString   = "random-state-string-change-in-production"
	token              *oauth2.Token
)

func init() {
	// Charger les variables d'environnement
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURI := os.Getenv("REDIRECT_URI")

	if clientID == "" || clientSecret == "" {
		log.Fatal("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	spotifyOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes: []string{
			"user-read-playback-state",
			"user-modify-playback-state",
			"user-read-currently-playing",
			"user-read-email",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}

	log.Printf("✅ Configuration loaded:")
	log.Printf("   Client ID: %s", clientID)
	log.Printf("   Redirect URI: %s", redirectURI)
}

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/player", handlePlayerStatus)
	http.HandleFunc("/play", handlePlay)
	http.HandleFunc("/pause", handlePause)
	http.HandleFunc("/queue", handleQueue)

	fmt.Println("🎵 Spike Spotify API - Server started on http://127.0.0.1:8080")
	fmt.Println("📍 Navigate to http://127.0.0.1:8080/login to authenticate")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>boeuf Spotify Spike</title></head>
	<body>
		<h1>🎵 boeuf - Spotify API Spike</h1>
		<h2>Tests OAuth & Player Control</h2>
		<ul>
			<li><a href="/login">1. Login with Spotify</a></li>
			<li><a href="/player">2. Get Player Status</a></li>
			<li><a href="/play">3. Play</a></li>
			<li><a href="/pause">4. Pause</a></li>
			<li><a href="/queue">5. Get Queue</a></li>
		</ul>
	</body>
	</html>
	`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	url := spotifyOAuthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		fmt.Fprintf(w, "Invalid state: %s", state)
		return
	}

	code := r.FormValue("code")
	var err error
	token, err = spotifyOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Fprintf(w, "Token exchange error: %v", err)
		return
	}

	fmt.Fprintf(w, "✅ Authentication successful!\n\n")
	fmt.Fprintf(w, "Access Token: %s...\n", token.AccessToken[:20])
	fmt.Fprintf(w, "Token Type: %s\n", token.TokenType)
	fmt.Fprintf(w, "Expiry: %s\n\n", token.Expiry)
	fmt.Fprintf(w, "<a href='/'>Go back to tests</a>")
}

func handlePlayerStatus(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		http.Error(w, "Not authenticated. Go to /login first", http.StatusUnauthorized)
		return
	}

	client := spotifyOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://api.spotify.com/v1/me/player")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		fmt.Fprintf(w, "⚠️  No active device found. Open Spotify on a device first.\n")
		return
	}

	if resp.StatusCode != 200 {
		fmt.Fprintf(w, "❌ Error: Status %d\n", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(result)
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		http.Error(w, "Not authenticated. Go to /login first", http.StatusUnauthorized)
		return
	}

	client := spotifyOAuthConfig.Client(context.Background(), token)
	req, _ := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/play", nil)
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		fmt.Fprintf(w, "✅ Play command sent successfully\n")
	} else if resp.StatusCode == 404 {
		fmt.Fprintf(w, "⚠️  No active device found. Open Spotify on a device first.\n")
	} else {
		fmt.Fprintf(w, "❌ Error: Status %d\n", resp.StatusCode)
	}
}

func handlePause(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		http.Error(w, "Not authenticated. Go to /login first", http.StatusUnauthorized)
		return
	}

	client := spotifyOAuthConfig.Client(context.Background(), token)
	req, _ := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/pause", nil)
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		fmt.Fprintf(w, "✅ Pause command sent successfully\n")
	} else if resp.StatusCode == 404 {
		fmt.Fprintf(w, "⚠️  No active device found. Open Spotify on a device first.\n")
	} else {
		fmt.Fprintf(w, "❌ Error: Status %d\n", resp.StatusCode)
	}
}

func handleQueue(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		http.Error(w, "Not authenticated. Go to /login first", http.StatusUnauthorized)
		return
	}

	client := spotifyOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://api.spotify.com/v1/me/player/queue")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintf(w, "❌ Error: Status %d\n", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(result)
}
