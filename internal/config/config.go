package config

import (
	"os"
	"strconv"
)

// Config holds server configuration loaded from environment variables.
type Config struct {
	Port       string
	DBPath     string
	MaxRooms   int
	MaxHistory int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		Port:       envOrDefault("PORT", "8080"),
		DBPath:     envOrDefault("DB_PATH", "chatterbox.db"),
		MaxRooms:   envOrDefaultInt("MAX_ROOMS", 100),
		MaxHistory: envOrDefaultInt("MAX_HISTORY", 50),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
