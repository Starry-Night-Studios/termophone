package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Contact struct {
	Name   string `json:"name"`
	PeerID string `json:"peer_id"`
}

type Config struct {
	Username        string    `json:"username"`
	Contacts        []Contact `json:"contacts"`
	AECTrimOffsetMs int       `json:"aec_trim_offset_ms"`
	ColorScheme     int       `json:"color_scheme"`
	ScreenQuality   string    `json:"screen_quality"`
}

var (
	mu      sync.Mutex
	current *Config
)

func getPath() string {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".termophone")
	os.MkdirAll(baseDir, 0755)
	return filepath.Join(baseDir, "config.json")
}

func Get() *Config {
	mu.Lock()
	defer mu.Unlock()

	if current != nil {
		return current
	}

	path := getPath()
	current = &Config{
		Username:        "Anon",
		Contacts:        []Contact{},
		AECTrimOffsetMs: 120, // Default hardware delay offset
		ColorScheme:     0,
		ScreenQuality:   "medium",
	}
	file, err := os.Open(path)
	if err == nil {
		json.NewDecoder(file).Decode(current)
		file.Close()
	} else {
		// Just write default, no more mapping old .termophone.json paths!
		writeToDisk(current)
	}
	return current
}

func writeToDisk(cfg *Config) {
	path := getPath()
	f, _ := os.Create(path)
	json.NewEncoder(f).Encode(cfg)
	f.Close()
}

func SaveContact(c Contact) {
	mu.Lock()
	defer mu.Unlock()

	if current == nil {
		return
	}

	for _, existing := range current.Contacts {
		if existing.PeerID == c.PeerID {
			return // already exists
		}
	}

	current.Contacts = append(current.Contacts, c)
	writeToDisk(current)
}

func RemoveContact(peerID string) {
	mu.Lock()
	defer mu.Unlock()

	if current == nil {
		return
	}

	filtered := make([]Contact, 0, len(current.Contacts))
	for _, c := range current.Contacts {
		if c.PeerID != peerID {
			filtered = append(filtered, c)
		}
	}

	current.Contacts = filtered
	writeToDisk(current)
}

func SaveConfig() {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		writeToDisk(current)
	}
}
