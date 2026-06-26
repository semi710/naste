package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds server configuration loaded from environment variables.
type Config struct {
	Port         string
	DataDir      string
	PrivateUser  string
	PrivatePass  string
	MaxPasteSize int64
}

// Load reads configuration from environment variables with sensible defaults.
// Supports PRIVATE_USER_FILE / PRIVATE_PASS_FILE for secret-file injection
// (e.g. sops-nix); file vars take precedence over inline vars.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data/paste"
	}

	user := os.Getenv("PRIVATE_USER")
	if f := os.Getenv("PRIVATE_USER_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			user = strings.TrimSpace(string(b))
		} else {
			log.Printf("warning: PRIVATE_USER_FILE %s: %v", f, err)
		}
	}

	pass := os.Getenv("PRIVATE_PASS")
	if f := os.Getenv("PRIVATE_PASS_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			pass = strings.TrimSpace(string(b))
		} else {
			log.Printf("warning: PRIVATE_PASS_FILE %s: %v", f, err)
		}
	}

	maxSize := int64(10 * 1024 * 1024) // 10 MB default
	if v := os.Getenv("MAX_PASTE_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxSize = n
		}
	}

	return &Config{
		Port:         port,
		DataDir:      dataDir,
		PrivateUser:  user,
		PrivatePass:  pass,
		MaxPasteSize: maxSize,
	}
}
