package telegram

import "testing"

func TestAllowPrivateChatOnly(t *testing.T) {
	t.Parallel()

	cfg := AccessConfig{
		AllowedChatTypes: []string{"private"},
		AllowedChatIDs:   []int64{1001},
	}

	if err := CheckAccess(cfg, 1001, 2001, "private"); err != nil {
		t.Fatalf("expected private chat to pass allowlist, got %v", err)
	}

	if err := CheckAccess(cfg, 9999, 2001, "group"); err == nil {
		t.Fatal("expected non-private chat to be rejected")
	}
}

func TestCheckAccessRequiresBothChatAndUserWhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := AccessConfig{
		AllowedChatTypes: []string{"private"},
		AllowedChatIDs:   []int64{1001},
		AllowedUserIDs:   []int64{2001},
	}

	if err := CheckAccess(cfg, 1001, 2001, "private"); err != nil {
		t.Fatalf("expected matching chat/user pair to pass, got %v", err)
	}

	if err := CheckAccess(cfg, 1001, 9999, "private"); err == nil {
		t.Fatal("expected mismatched user to be rejected")
	}

	if err := CheckAccess(cfg, 9999, 2001, "private"); err == nil {
		t.Fatal("expected mismatched chat to be rejected")
	}
}

func TestCheckAccessAllowsConfiguredGroupChatTypes(t *testing.T) {
	t.Parallel()

	cfg := AccessConfig{
		AllowedChatTypes: []string{"private", "group", "supergroup"},
		AllowedChatIDs:   []int64{1001},
	}

	if err := CheckAccess(cfg, 1001, 2001, "group"); err != nil {
		t.Fatalf("expected group chat to pass allowlist, got %v", err)
	}
	if err := CheckAccess(cfg, 1001, 2001, "supergroup"); err != nil {
		t.Fatalf("expected supergroup chat to pass allowlist, got %v", err)
	}
}

func TestCheckAccessRejectsWhenNoAllowlistIsConfigured(t *testing.T) {
	t.Parallel()

	cfg := AccessConfig{
		AllowedChatTypes: []string{"private"},
	}

	if err := CheckAccess(cfg, 1001, 2001, "private"); err == nil {
		t.Fatal("expected empty allowlists to deny access")
	}
}
