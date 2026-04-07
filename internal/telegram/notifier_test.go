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
}
