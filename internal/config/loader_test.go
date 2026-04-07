package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesTelegramNotificationDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.Telegram.NotifyOnAccept {
		t.Fatal("expected notify_on_accept default to be true")
	}
	if !cfg.Telegram.NotifyOnComplete {
		t.Fatal("expected notify_on_complete default to be true")
	}
	if !cfg.Telegram.NotifyOnFail {
		t.Fatal("expected notify_on_fail default to be true")
	}
	if cfg.Telegram.WebhookURL != "" {
		t.Fatalf("expected webhook_url default to be empty, got %q", cfg.Telegram.WebhookURL)
	}
	if cfg.Telegram.WebhookSecret != "" {
		t.Fatalf("expected webhook_secret default to be empty, got %q", cfg.Telegram.WebhookSecret)
	}
}
