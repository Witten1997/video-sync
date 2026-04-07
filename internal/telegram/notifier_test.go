package telegram

import "testing"

func TestBuildCompletionReplyUsesSingleMessageUpdate(t *testing.T) {
	t.Parallel()

	reply := BuildStatusReply(StatusReplyInput{
		Stage:     TelegramRequestStatusCompleted,
		Title:     "demo video",
		TaskID:    "task-1",
		RecordID:  99,
		MessageID: 123,
	})

	if reply.EditMessageID != 123 {
		t.Fatalf("expected edit-in-place reply, got %+v", reply)
	}
	if reply.Text == "" {
		t.Fatal("expected non-empty completion text")
	}
	if reply.Text != "下载完成。\n标题：demo video\n记录 ID：99" {
		t.Fatalf("expected chinese completion reply, got %q", reply.Text)
	}
}
