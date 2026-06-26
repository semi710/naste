package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.DataDir != "/data/paste" {
		t.Errorf("DataDir = %q, want /data/paste", cfg.DataDir)
	}
	if cfg.MaxPasteSize != 10*1024*1024 {
		t.Errorf("MaxPasteSize = %d, want %d", cfg.MaxPasteSize, 10*1024*1024)
	}
	if cfg.PrivateUser != "" || cfg.PrivatePass != "" {
		t.Error("expected empty private creds by default")
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Clearenv()
	t.Setenv("PORT", "9090")
	t.Setenv("DATA_DIR", "/tmp/naste")
	t.Setenv("PRIVATE_USER", "admin")
	t.Setenv("PRIVATE_PASS", "secret")
	t.Setenv("MAX_PASTE_SIZE", "2048")

	cfg := Load()
	if cfg.Port != "9090" || cfg.DataDir != "/tmp/naste" {
		t.Error("env override failed")
	}
	if cfg.PrivateUser != "admin" || cfg.PrivatePass != "secret" {
		t.Error("private creds not loaded")
	}
	if cfg.MaxPasteSize != 2048 {
		t.Errorf("MaxPasteSize = %d, want 2048", cfg.MaxPasteSize)
	}
}

func TestMaxPasteSizeInvalid(t *testing.T) {
	os.Clearenv()
	t.Setenv("MAX_PASTE_SIZE", "not-a-number")
	cfg := Load()
	if cfg.MaxPasteSize != 10*1024*1024 {
		t.Error("invalid MAX_PASTE_SIZE should fall back to default")
	}
}

func TestSecretFiles(t *testing.T) {
	os.Clearenv()
	dir := t.TempDir()
	userFile := filepath.Join(dir, "user")
	passFile := filepath.Join(dir, "pass")
	if err := os.WriteFile(userFile, []byte("  fileuser  "), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passFile, []byte("  filepass\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PRIVATE_USER", "envuser")
	t.Setenv("PRIVATE_USER_FILE", userFile)
	t.Setenv("PRIVATE_PASS", "envpass")
	t.Setenv("PRIVATE_PASS_FILE", passFile)

	cfg := Load()
	if cfg.PrivateUser != "fileuser" {
		t.Errorf("PrivateUser = %q, want fileuser (file should override env)", cfg.PrivateUser)
	}
	if cfg.PrivatePass != "filepass" {
		t.Errorf("PrivatePass = %q, want filepass (file should override env)", cfg.PrivatePass)
	}
}
