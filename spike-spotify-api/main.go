package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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
	http.HandleFunc("/devices", handleDevices)
	http.HandleFunc("/play", handlePlay)
	http.HandleFunc("/pause", handlePause)
	http.HandleFunc("/queue", handleQueue)
	http.HandleFunc("/loop", handleLoop)

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
			<li><a href="/devices">3. Get Devices</a></li>
			<li><a href="/play">4. Play</a></li>
			<li><a href="/pause">5. Pause</a></li>
			<li><a href="/queue">6. Get Queue</a></li>
			<li><a href="/loop?count=3&delay_ms=1000">7. Loop Play/Pause</a></li>
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
	respData, err := doSpotifyRequest(client, "PUT", "https://api.spotify.com/v1/me/player/play")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	if respData.StatusCode == 204 {
		fmt.Fprintf(w, "✅ Play command sent successfully\n")
	} else if respData.StatusCode == 404 {
		fmt.Fprintf(w, "⚠️  No active device found. Open Spotify on a device first.\n")
	} else {
		fmt.Fprintf(w, "❌ Error: Status %d\n", respData.StatusCode)
	}

	if respData.ContentType != "" {
		fmt.Fprintf(w, "Content-Type: %s\n", respData.ContentType)
	}
	if len(respData.Body) > 0 {
		fmt.Fprintf(w, "Response Body: %s\n", respData.Body)
	}

	logAction("play", respData, client)
}

func handlePause(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		http.Error(w, "Not authenticated. Go to /login first", http.StatusUnauthorized)
		return
	}

	client := spotifyOAuthConfig.Client(context.Background(), token)
	respData, err := doSpotifyRequest(client, "PUT", "https://api.spotify.com/v1/me/player/pause")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	if respData.StatusCode == 204 {
		fmt.Fprintf(w, "✅ Pause command sent successfully\n")
	} else if respData.StatusCode == 404 {
		fmt.Fprintf(w, "⚠️  No active device found. Open Spotify on a device first.\n")
	} else {
		fmt.Fprintf(w, "❌ Error: Status %d\n", respData.StatusCode)
	}

	if respData.ContentType != "" {
		fmt.Fprintf(w, "Content-Type: %s\n", respData.ContentType)
	}
	if len(respData.Body) > 0 {
		fmt.Fprintf(w, "Response Body: %s\n", respData.Body)
	}

	logAction("pause", respData, client)
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

func handleDevices(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		http.Error(w, "Not authenticated. Go to /login first", http.StatusUnauthorized)
		return
	}

	client := spotifyOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://api.spotify.com/v1/me/player/devices")
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

func handleLoop(w http.ResponseWriter, r *http.Request) {
	if token == nil {
		http.Error(w, "Not authenticated. Go to /login first", http.StatusUnauthorized)
		return
	}

	count := parseQueryInt(r, "count", 3)
	delayMs := parseQueryInt(r, "delay_ms", 1000)
	if count <= 0 {
		count = 1
	}
	if delayMs < 0 {
		delayMs = 0
	}

	client := spotifyOAuthConfig.Client(context.Background(), token)

	for i := 1; i <= count; i++ {
		pauseResp, err := doSpotifyRequest(client, "PUT", "https://api.spotify.com/v1/me/player/pause")
		if err == nil {
			logAction("pause", pauseResp, client)
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)

		playResp, err := doSpotifyRequest(client, "PUT", "https://api.spotify.com/v1/me/player/play")
		if err == nil {
			logAction("play", playResp, client)
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	fmt.Fprintf(w, "✅ Loop completed. Logged %d pause/play cycles to spike-player-actions.log\n", count)
}

type spotifyResponse struct {
	StatusCode  int
	ContentType string
	Body        string
}

func doSpotifyRequest(client *http.Client, method, url string) (spotifyResponse, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return spotifyResponse{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return spotifyResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	return spotifyResponse{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        bodyStr,
	}, nil
}

type playerState struct {
	StatusCode int
	IsPlaying  *bool
}

type devicesSnapshot struct {
	StatusCode int
	Summary    string
}

func logAction(action string, resp spotifyResponse, client *http.Client) {
	state := fetchPlayerState(client)
	devices := fetchDevicesSummary(client)

	line := fmt.Sprintf(
		"%s action=%s status=%d content_type=%q body=%q is_playing=%s devices=%q\n",
		time.Now().Format(time.RFC3339),
		action,
		resp.StatusCode,
		resp.ContentType,
		resp.Body,
		formatIsPlaying(state),
		devices.Summary,
	)

	appendLogLine("spike-player-actions.log", line)
}

func fetchPlayerState(client *http.Client) playerState {
	resp, err := client.Get("https://api.spotify.com/v1/me/player")
	if err != nil {
		return playerState{StatusCode: 0, IsPlaying: nil}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return playerState{StatusCode: resp.StatusCode, IsPlaying: nil}
	}

	var result struct {
		IsPlaying bool `json:"is_playing"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return playerState{StatusCode: resp.StatusCode, IsPlaying: nil}
	}

	return playerState{StatusCode: resp.StatusCode, IsPlaying: &result.IsPlaying}
}

func fetchDevicesSummary(client *http.Client) devicesSnapshot {
	resp, err := client.Get("https://api.spotify.com/v1/me/player/devices")
	if err != nil {
		return devicesSnapshot{StatusCode: 0, Summary: "request_error"}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return devicesSnapshot{StatusCode: resp.StatusCode, Summary: fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(bodyBytes))}
	}

	var payload struct {
		Devices []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Type         string `json:"type"`
			IsActive     bool   `json:"is_active"`
			IsRestricted bool   `json:"is_restricted"`
		} `json:"devices"`
	}

	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return devicesSnapshot{StatusCode: resp.StatusCode, Summary: "unparseable"}
	}

	summary := ""
	for i, device := range payload.Devices {
		if i > 0 {
			summary += "; "
		}
		summary += fmt.Sprintf("%s(id=%s,active=%t,restricted=%t,type=%s)", device.Name, device.ID, device.IsActive, device.IsRestricted, device.Type)
	}

	if summary == "" {
		summary = "no_devices"
	}

	return devicesSnapshot{StatusCode: resp.StatusCode, Summary: summary}
}

func appendLogLine(path, line string) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("log write error: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(line); err != nil {
		log.Printf("log write error: %v", err)
	}
}

func parseQueryInt(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func formatIsPlaying(state playerState) string {
	if state.IsPlaying == nil {
		return "unknown"
	}
	if *state.IsPlaying {
		return "true"
	}
	return "false"
}
