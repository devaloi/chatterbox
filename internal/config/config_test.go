package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DBPath != "chatterbox.db" {
		t.Errorf("expected default db path chatterbox.db, got %s", cfg.DBPath)
	}
	if cfg.MaxRooms != 100 {
		t.Errorf("expected default max rooms 100, got %d", cfg.MaxRooms)
	}
	if cfg.MaxHistory != 50 {
		t.Errorf("expected default max history 50, got %d", cfg.MaxHistory)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", "/tmp/test.db")
	t.Setenv("MAX_ROOMS", "50")
	t.Setenv("MAX_HISTORY", "25")

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("expected db path /tmp/test.db, got %s", cfg.DBPath)
	}
	if cfg.MaxRooms != 50 {
		t.Errorf("expected max rooms 50, got %d", cfg.MaxRooms)
	}
	if cfg.MaxHistory != 25 {
		t.Errorf("expected max history 25, got %d", cfg.MaxHistory)
	}
}

func TestLoadInvalidInt(t *testing.T) {
	os.Setenv("MAX_ROOMS", "notanumber")
	defer os.Unsetenv("MAX_ROOMS")

	cfg := Load()
	if cfg.MaxRooms != 100 {
		t.Errorf("expected fallback max rooms 100, got %d", cfg.MaxRooms)
	}
}
