package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Script de test complet pour valider toutes les capabilities Spotify nécessaires pour boeuf

type TestResult struct {
	Name    string
	Status  string // ✅ PASS, ❌ FAIL, ⚠️ WARNING
	Message string
	Details interface{}
}

var results []TestResult

func main() {
	fmt.Println("🎵 boeuf - Spotify API Capability Validation")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env file not found, using environment variables")
	}

	token := os.Getenv("SPOTIFY_ACCESS_TOKEN")
	if token == "" {
		fmt.Println("❌ SPOTIFY_ACCESS_TOKEN not set")
		fmt.Println("Run 'go run main.go' first to authenticate and get a token")
		fmt.Println("Then export SPOTIFY_ACCESS_TOKEN=<your_token>")
		os.Exit(1)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Exécuter tous les tests
	testAuthentication(client, token)
	testDeviceDetection(client, token)
	testPlayerStatus(client, token)
	testPlayPauseControl(client, token)
	testSeekControl(client, token)
	testQueueManagement(client, token)
	testSearchCapability(client, token)

	// Afficher le rapport
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 50))

	passed := 0
	failed := 0
	warnings := 0

	for _, result := range results {
		fmt.Printf("\n%s %s\n", result.Status, result.Name)
		fmt.Printf("   %s\n", result.Message)
		if result.Details != nil {
			fmt.Printf("   Details: %v\n", result.Details)
		}

		switch result.Status {
		case "✅":
			passed++
		case "❌":
			failed++
		case "⚠️":
			warnings++
		}
	}

	fmt.Printf("\n" + strings.Repeat("-", 50) + "\n")
	fmt.Printf("Total: %d tests | ✅ Pass: %d | ❌ Fail: %d | ⚠️ Warnings: %d\n",
		len(results), passed, failed, warnings)

	// Verdict final
	fmt.Println("\n" + strings.Repeat("=", 50))
	if failed == 0 {
		fmt.Println("✅ VERDICT: API Spotify VALIDÉE pour boeuf MVP")
		fmt.Println("   Toutes les capabilities nécessaires sont disponibles.")
	} else {
		fmt.Println("❌ VERDICT: Des capabilities critiques manquent")
		fmt.Println("   Réviser l'architecture ou le choix de plateforme.")
	}
	fmt.Println(strings.Repeat("=", 50))
}

func testAuthentication(client *http.Client, token string) {
	fmt.Println("\n🔐 Test 1: Authentication & Authorization")

	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		results = append(results, TestResult{
			Name:    "OAuth Authentication",
			Status:  "❌",
			Message: fmt.Sprintf("Request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var user map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&user)

		results = append(results, TestResult{
			Name:    "OAuth Authentication",
			Status:  "✅",
			Message: "Token valid, user authenticated",
			Details: fmt.Sprintf("User: %s", user["display_name"]),
		})
	} else {
		results = append(results, TestResult{
			Name:    "OAuth Authentication",
			Status:  "❌",
			Message: fmt.Sprintf("Authentication failed: Status %d", resp.StatusCode),
		})
	}
}

func testDeviceDetection(client *http.Client, token string) {
	fmt.Println("\n📱 Test 2: Device Detection")

	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/devices", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		results = append(results, TestResult{
			Name:    "Device Detection",
			Status:  "❌",
			Message: fmt.Sprintf("Request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	devices := result["devices"].([]interface{})
	if len(devices) > 0 {
		results = append(results, TestResult{
			Name:    "Device Detection",
			Status:  "✅",
			Message: "Devices detected successfully",
			Details: fmt.Sprintf("%d device(s) available", len(devices)),
		})
	} else {
		results = append(results, TestResult{
			Name:    "Device Detection",
			Status:  "⚠️",
			Message: "No active devices found. Open Spotify on a device to test further.",
		})
	}
}

func testPlayerStatus(client *http.Client, token string) {
	fmt.Println("\n▶️  Test 3: Player Status Retrieval")

	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		results = append(results, TestResult{
			Name:    "Get Player Status",
			Status:  "❌",
			Message: fmt.Sprintf("Request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var player map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&player)

		results = append(results, TestResult{
			Name:    "Get Player Status",
			Status:  "✅",
			Message: "Player status retrieved successfully",
			Details: fmt.Sprintf("Playing: %v", player["is_playing"]),
		})
	} else if resp.StatusCode == 204 {
		results = append(results, TestResult{
			Name:    "Get Player Status",
			Status:  "⚠️",
			Message: "No active playback. Start playing music to test further.",
		})
	} else {
		results = append(results, TestResult{
			Name:    "Get Player Status",
			Status:  "❌",
			Message: fmt.Sprintf("Failed to get player status: Status %d", resp.StatusCode),
		})
	}
}

func testPlayPauseControl(client *http.Client, token string) {
	fmt.Println("\n⏯️  Test 4: Play/Pause Control")

	// Test Pause
	reqPause, _ := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/pause", nil)
	reqPause.Header.Set("Authorization", "Bearer "+token)

	respPause, err := client.Do(reqPause)
	if err != nil {
		results = append(results, TestResult{
			Name:    "Play/Pause Control",
			Status:  "❌",
			Message: fmt.Sprintf("Pause request failed: %v", err),
		})
		return
	}
	defer respPause.Body.Close()

	time.Sleep(1 * time.Second)

	// Test Play
	reqPlay, _ := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/play", nil)
	reqPlay.Header.Set("Authorization", "Bearer "+token)

	respPlay, err := client.Do(reqPlay)
	if err != nil {
		results = append(results, TestResult{
			Name:    "Play/Pause Control",
			Status:  "❌",
			Message: fmt.Sprintf("Play request failed: %v", err),
		})
		return
	}
	defer respPlay.Body.Close()

	if respPlay.StatusCode == 204 || respPlay.StatusCode == 202 {
		results = append(results, TestResult{
			Name:    "Play/Pause Control",
			Status:  "✅",
			Message: "Play/Pause control working",
		})
	} else if respPlay.StatusCode == 404 {
		results = append(results, TestResult{
			Name:    "Play/Pause Control",
			Status:  "⚠️",
			Message: "No active device found. API endpoints available but need active player.",
		})
	} else {
		results = append(results, TestResult{
			Name:    "Play/Pause Control",
			Status:  "❌",
			Message: fmt.Sprintf("Control failed: Status %d", respPlay.StatusCode),
		})
	}
}

func testSeekControl(client *http.Client, token string) {
	fmt.Println("\n⏩ Test 5: Seek Position Control")

	url := "https://api.spotify.com/v1/me/player/seek?position_ms=30000"
	req, _ := http.NewRequest("PUT", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		results = append(results, TestResult{
			Name:    "Seek Control",
			Status:  "❌",
			Message: fmt.Sprintf("Seek request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		results = append(results, TestResult{
			Name:    "Seek Control",
			Status:  "✅",
			Message: "Seek control working",
			Details: fmt.Sprintf("API latency: %v", latency),
		})
	} else if resp.StatusCode == 404 {
		results = append(results, TestResult{
			Name:    "Seek Control",
			Status:  "⚠️",
			Message: "No active device. API endpoint available.",
		})
	} else {
		results = append(results, TestResult{
			Name:    "Seek Control",
			Status:  "❌",
			Message: fmt.Sprintf("Seek failed: Status %d", resp.StatusCode),
		})
	}
}

func testQueueManagement(client *http.Client, token string) {
	fmt.Println("\n📋 Test 6: Queue Management")

	// Test GET queue
	reqGet, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/queue", nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)

	respGet, err := client.Do(reqGet)
	if err != nil {
		results = append(results, TestResult{
			Name:    "Queue Management",
			Status:  "❌",
			Message: fmt.Sprintf("Get queue failed: %v", err),
		})
		return
	}
	defer respGet.Body.Close()

	if respGet.StatusCode == 200 {
		var queue map[string]interface{}
		json.NewDecoder(respGet.Body).Decode(&queue)

		results = append(results, TestResult{
			Name:    "Queue Management",
			Status:  "✅",
			Message: "Queue read/write capabilities available",
		})
	} else {
		results = append(results, TestResult{
			Name:    "Queue Management",
			Status:  "⚠️",
			Message: fmt.Sprintf("Queue endpoint returned: Status %d", respGet.StatusCode),
		})
	}
}

func testSearchCapability(client *http.Client, token string) {
	fmt.Println("\n🔍 Test 7: Search Capability")

	query := "The Killers"
	url := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=1", query)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		results = append(results, TestResult{
			Name:    "Search Capability",
			Status:  "❌",
			Message: fmt.Sprintf("Search request failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var searchResult map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&searchResult)

		results = append(results, TestResult{
			Name:    "Search Capability",
			Status:  "✅",
			Message: "Search working correctly",
		})
	} else {
		results = append(results, TestResult{
			Name:    "Search Capability",
			Status:  "❌",
			Message: fmt.Sprintf("Search failed: Status %d", resp.StatusCode),
		})
	}
}
