package config

import (
	"os"
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
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data/paste"
	}

	maxSize := int64(10 * 1024 * 1024) // 10 MB

	return &Config{
		Port:         port,
		DataDir:      dataDir,
		PrivateUser:  os.Getenv("PRIVATE_USER"),
		PrivatePass:  os.Getenv("PRIVATE_PASS"),
		MaxPasteSize: maxSize,
	}
}
