package telegram

import "testing"

func TestExtractURLsSupportsDirectMessageAndCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "direct url message",
			text: "https://youtu.be/demo",
			want: []string{"https://youtu.be/demo"},
		},
		{
			name: "download command",
			text: "/download https://www.bilibili.com/video/BV1xx411c7mD",
			want: []string{"https://www.bilibili.com/video/BV1xx411c7mD"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ExtractURLs(tt.text, 1)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d urls, got %d (%v)", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("expected url %q at index %d, got %q", tt.want[i], i, got[i])
				}
			}
		})
	}
}

func TestParseMessageRejectsMultipleURLs(t *testing.T) {
	t.Parallel()

	result := ParseMessage("https://a.example https://b.example", 1)
	if result.Kind != ParseResultKindReject {
		t.Fatalf("expected reject result, got %q", result.Kind)
	}
	if result.ReplyText == "" {
		t.Fatal("expected reject reply text")
	}
}

func TestParseMessageIgnoresUnsupportedText(t *testing.T) {
	t.Parallel()

	result := ParseMessage("hello world", 1)
	if result.Kind != ParseResultKindIgnore {
		t.Fatalf("expected ignore result, got %q", result.Kind)
	}
}

func TestParseMessageRequiresBareDirectURL(t *testing.T) {
	t.Parallel()

	result := ParseMessage("please download https://example.com/video", 1)
	if result.Kind != ParseResultKindIgnore {
		t.Fatalf("expected ignore result for non-bare direct URL, got %q", result.Kind)
	}
}

func TestParseMessageIgnoresBotMentionCommandWithoutGroupContext(t *testing.T) {
	t.Parallel()

	result := ParseMessage("/download@mybot https://example.com/video", 1)
	if result.Kind != ParseResultKindIgnore {
		t.Fatalf("expected ignore result without explicit group context, got %q", result.Kind)
	}
}

func TestParseMessageSupportsStatusCommand(t *testing.T) {
	t.Parallel()

	result := ParseMessage("/status task-123", 1)
	if result.Kind != ParseResultKindStatus {
		t.Fatalf("expected status result, got %q", result.Kind)
	}
	if result.TaskID != "task-123" {
		t.Fatalf("expected task id task-123, got %q", result.TaskID)
	}
}

func TestParseMessageSupportsHelpCommand(t *testing.T) {
	t.Parallel()

	result := ParseMessage("/help", 1)
	if result.Kind != ParseResultKindHelp {
		t.Fatalf("expected help result, got %q", result.Kind)
	}
	if result.ReplyText == "" {
		t.Fatal("expected help reply text")
	}
}

func TestParseMessageForChatAcceptsMentionedGroupCommand(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("/download@mybot https://example.com/video", 1, "group", "mybot")
	if result.Kind != ParseResultKindSubmit {
		t.Fatalf("expected submit result, got %q", result.Kind)
	}
	if result.URL != "https://example.com/video" {
		t.Fatalf("expected parsed url, got %q", result.URL)
	}
}

func TestParseMessageForChatAcceptsMentionedGroupStatusCommand(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("/status@mybot task-123", 1, "group", "mybot")
	if result.Kind != ParseResultKindStatus {
		t.Fatalf("expected status result, got %q", result.Kind)
	}
	if result.TaskID != "task-123" {
		t.Fatalf("expected task id task-123, got %q", result.TaskID)
	}
}

func TestParseMessageForChatAcceptsMentionedGroupHelpCommand(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("/help@mybot", 1, "group", "mybot")
	if result.Kind != ParseResultKindHelp {
		t.Fatalf("expected help result, got %q", result.Kind)
	}
}

func TestParseMessageForChatIgnoresGroupCommandWithoutMention(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("/download https://example.com/video", 1, "group", "mybot")
	if result.Kind != ParseResultKindIgnore {
		t.Fatalf("expected ignore result for unmentioned group command, got %q", result.Kind)
	}
}

func TestParseMessageForChatAcceptsLeadingMentionDirectURL(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("@mybot https://example.com/video", 1, "supergroup", "mybot")
	if result.Kind != ParseResultKindSubmit {
		t.Fatalf("expected submit result for mentioned group url, got %q", result.Kind)
	}
	if result.URL != "https://example.com/video" {
		t.Fatalf("expected parsed url, got %q", result.URL)
	}
}

func TestParseMessageForChatAcceptsLeadingMentionStatus(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("@mybot /status task-123", 1, "group", "mybot")
	if result.Kind != ParseResultKindStatus {
		t.Fatalf("expected status result for mentioned group command, got %q", result.Kind)
	}
	if result.TaskID != "task-123" {
		t.Fatalf("expected task id task-123, got %q", result.TaskID)
	}
}

func TestParseMessageForChatAcceptsLeadingMentionHelp(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("@mybot /help", 1, "group", "mybot")
	if result.Kind != ParseResultKindHelp {
		t.Fatalf("expected help result for mentioned group command, got %q", result.Kind)
	}
}

func TestParseMessageUsesChineseRejectText(t *testing.T) {
	t.Parallel()

	result := ParseMessage("/download", 1)
	if result.Kind != ParseResultKindReject {
		t.Fatalf("expected reject result, got %q", result.Kind)
	}
	if result.ReplyText != "用法：/download <url>" {
		t.Fatalf("expected chinese usage text, got %q", result.ReplyText)
	}

	result = ParseMessage("https://a.example https://b.example", 1)
	if result.ReplyText != "每条消息只支持一个 URL。" {
		t.Fatalf("expected chinese multi-url reject text, got %q", result.ReplyText)
	}
}

func TestParseMessageForChatIgnoresWrongBotMention(t *testing.T) {
	t.Parallel()

	result := ParseMessageForChat("/download@otherbot https://example.com/video", 1, "group", "mybot")
	if result.Kind != ParseResultKindIgnore {
		t.Fatalf("expected ignore result for other bot command, got %q", result.Kind)
	}

	result = ParseMessageForChat("@otherbot https://example.com/video", 1, "group", "mybot")
	if result.Kind != ParseResultKindIgnore {
		t.Fatalf("expected ignore result for other bot mention, got %q", result.Kind)
	}
}
