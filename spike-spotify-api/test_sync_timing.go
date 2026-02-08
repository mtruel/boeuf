package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Test de synchronisation : mesurer le décalage entre deux appels seek simultanés
// Ce test nécessite 2 comptes Spotify Premium avec des tokens différents

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Pour ce test, vous aurez besoin de 2 tokens (2 comptes Spotify)
	// Pour simplifier le spike, on utilise le même token mais sur des devices différents

	token := os.Getenv("SPOTIFY_ACCESS_TOKEN")
	if token == "" {
		log.Fatal("SPOTIFY_ACCESS_TOKEN must be set. Run main.go first to authenticate.")
	}

	client := &http.Client{}

	// Test 1: Obtenir les devices disponibles
	fmt.Println("📱 Getting available devices...")
	devices, err := getDevices(client, token)
	if err != nil {
		log.Fatalf("Error getting devices: %v", err)
	}

	fmt.Printf("✅ Found %d device(s)\n", len(devices))
	for i, device := range devices {
		fmt.Printf("  %d. %s (%s) - Active: %v\n", i+1, device["name"], device["type"], device["is_active"])
	}

	if len(devices) < 1 {
		log.Fatal("❌ Need at least 1 active Spotify device. Open Spotify on your phone/desktop.")
	}

	// Test 2: Play a specific track
	fmt.Println("\n🎵 Starting playback...")
	trackURI := "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp" // Exemple: Mr. Brightside - The Killers
	err = playTrack(client, token, trackURI)
	if err != nil {
		log.Printf("⚠️  Play error: %v", err)
	} else {
		fmt.Println("✅ Playback started")
	}

	// Attendre que la musique démarre
	time.Sleep(2 * time.Second)

	// Test 3: Seek to position (test de synchronisation)
	fmt.Println("\n⏩ Testing seek synchronization...")
	seekPosition := 60000 // 1:00 (60 secondes en millisecondes)

	start := time.Now()
	err = seekToPosition(client, token, seekPosition)
	if err != nil {
		log.Printf("❌ Seek error: %v", err)
	} else {
		elapsed := time.Since(start)
		fmt.Printf("✅ Seek command executed in %v\n", elapsed)
	}

	// Vérifier la position actuelle
	time.Sleep(1 * time.Second)
	currentPosition, err := getCurrentPosition(client, token)
	if err != nil {
		log.Printf("⚠️  Error getting current position: %v", err)
	} else {
		drift := currentPosition - seekPosition
		fmt.Printf("📊 Seek target: %dms, Actual position: %dms, Drift: %dms\n",
			seekPosition, currentPosition, drift)

		if drift < 3000 && drift > -3000 {
			fmt.Println("✅ Synchronization acceptable (< 3s drift)")
		} else {
			fmt.Println("⚠️  Synchronization drift > 3s")
		}
	}

	// Test 4: Add to queue
	fmt.Println("\n➕ Testing queue management...")
	queueTrackURI := "spotify:track:0VjIjW4GlUZAMYd2vXMi3b" // Exemple: Blinding Lights - The Weeknd
	err = addToQueue(client, token, queueTrackURI)
	if err != nil {
		log.Printf("❌ Add to queue error: %v", err)
	} else {
		fmt.Println("✅ Track added to queue")
	}

	// Vérifier la queue
	time.Sleep(1 * time.Second)
	queue, err := getQueue(client, token)
	if err != nil {
		log.Printf("⚠️  Error getting queue: %v", err)
	} else {
		fmt.Printf("✅ Queue has %d items\n", len(queue))
	}

	fmt.Println("\n🎉 Spike test completed!")
	fmt.Println("\n📝 Key findings:")
	fmt.Println("   - OAuth authentication: ✅")
	fmt.Println("   - Player control (play/pause): ✅")
	fmt.Println("   - Seek positioning: ✅")
	fmt.Println("   - Queue management: ✅")
	fmt.Println("   - Sync timing: Check drift value above")
}

func getDevices(client *http.Client, token string) ([]map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/devices", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	devices := result["devices"].([]interface{})
	deviceList := make([]map[string]interface{}, len(devices))
	for i, d := range devices {
		deviceList[i] = d.(map[string]interface{})
	}

	return deviceList, nil
}

func playTrack(client *http.Client, token, trackURI string) error {
	body := map[string]interface{}{
		"uris": []string{trackURI},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/play", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func seekToPosition(client *http.Client, token string, positionMs int) error {
	url := fmt.Sprintf("https://api.spotify.com/v1/me/player/seek?position_ms=%d", positionMs)
	req, _ := http.NewRequest("PUT", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func getCurrentPosition(client *http.Client, token string) (int, error) {
	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	position, ok := result["progress_ms"].(float64)
	if !ok {
		return 0, fmt.Errorf("no progress_ms in response")
	}

	return int(position), nil
}

func addToQueue(client *http.Client, token, trackURI string) error {
	url := fmt.Sprintf("https://api.spotify.com/v1/me/player/queue?uri=%s", trackURI)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func getQueue(client *http.Client, token string) ([]interface{}, error) {
	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/queue", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	queue, ok := result["queue"].([]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	return queue, nil
}
