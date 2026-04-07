package config

import "testing"

func TestTelegramConfigValidateAllowsWebhookMode(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:            true,
		BotToken:           "123:token",
		Mode:               "webhook",
		WebhookURL:         "https://example.com/telegram/webhook",
		WebhookSecret:      "secret-123",
		AllowedChatTypes:   []string{"private"},
		MaxURLsPerMessage:  1,
		NotifyOnAccept:     true,
		NotifyOnComplete:   true,
		NotifyOnFail:       true,
		AllowedChatIDs:     []int64{1001},
		PollTimeoutSeconds: 30,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected webhook config to validate, got %v", err)
	}
}

func TestTelegramConfigValidateRequiresWebhookSecret(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:           true,
		BotToken:          "123:token",
		Mode:              "webhook",
		WebhookURL:        "https://example.com/telegram/webhook",
		AllowedChatTypes:  []string{"private"},
		MaxURLsPerMessage: 1,
		AllowedChatIDs:    []int64{1001},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected webhook mode without secret to fail validation")
	}
}

func TestTelegramConfigValidateRejectsNonHTTPSWebhookURL(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:           true,
		BotToken:          "123:token",
		Mode:              "webhook",
		WebhookURL:        "http://example.com/telegram/webhook",
		WebhookSecret:     "secret-123",
		AllowedChatTypes:  []string{"private"},
		MaxURLsPerMessage: 1,
		AllowedChatIDs:    []int64{1001},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-https webhook URL to fail validation")
	}
}

func TestTelegramConfigValidateRejectsInvalidWebhookSecret(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:           true,
		BotToken:          "123:token",
		Mode:              "webhook",
		WebhookURL:        "https://example.com/telegram/webhook",
		WebhookSecret:     "bad secret!",
		AllowedChatTypes:  []string{"private"},
		MaxURLsPerMessage: 1,
		AllowedChatIDs:    []int64{1001},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid webhook secret to fail validation")
	}
}

func TestTelegramConfigValidateRejectsHostlessHTTPSWebhookURL(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:           true,
		BotToken:          "123:token",
		Mode:              "webhook",
		WebhookURL:        "https://",
		WebhookSecret:     "secret-123",
		AllowedChatTypes:  []string{"private"},
		MaxURLsPerMessage: 1,
		AllowedChatIDs:    []int64{1001},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected hostless https webhook URL to fail validation")
	}
}

func TestTelegramConfigValidateAllowsGroupChatTypes(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:            true,
		BotToken:           "123:token",
		Mode:               "polling",
		PollTimeoutSeconds: 30,
		AllowedChatTypes:   []string{"private", "group", "supergroup"},
		MaxURLsPerMessage:  1,
		AllowedChatIDs:     []int64{1001},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected group-capable chat types to validate, got %v", err)
	}
}

func TestTelegramConfigValidateAllowsEmptyAllowlist(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:            true,
		BotToken:           "123:token",
		Mode:               "polling",
		PollTimeoutSeconds: 30,
		AllowedChatTypes:   []string{"private"},
		MaxURLsPerMessage:  1,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected empty allowlists to validate for pending approvals flow, got %v", err)
	}
}

func TestTelegramConfigValidateRejectsUnsupportedChatType(t *testing.T) {
	t.Parallel()

	cfg := TelegramConfig{
		Enabled:            true,
		BotToken:           "123:token",
		Mode:               "polling",
		PollTimeoutSeconds: 30,
		AllowedChatTypes:   []string{"channel"},
		MaxURLsPerMessage:  1,
		AllowedChatIDs:     []int64{1001},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsupported chat type to fail validation")
	}
}
