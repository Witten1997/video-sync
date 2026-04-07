package telegram

import (
	"fmt"
	"strings"
)

const (
	TelegramRequestStatusPending   = "pending"
	TelegramRequestStatusQueued    = "queued"
	TelegramRequestStatusCompleted = "completed"
	TelegramRequestStatusFailed    = "failed"
)

type StatusReplyInput struct {
	Stage        string
	Title        string
	TaskID       string
	RecordID     uint
	MessageID    int64
	ErrorMessage string
}

type StatusReply struct {
	EditMessageID int64
	Text          string
}

func BuildStatusReply(in StatusReplyInput) StatusReply {
	var text string

	switch in.Stage {
	case TelegramRequestStatusCompleted:
		text = "Download completed."
		if in.Title != "" {
			text += "\nTitle: " + in.Title
		}
		if in.RecordID > 0 {
			text += "\nRecord ID: " + fmt.Sprint(in.RecordID)
		}
	case TelegramRequestStatusFailed:
		text = "Download failed."
		if in.ErrorMessage != "" {
			text += "\nReason: " + in.ErrorMessage
		}
	default:
		text = "Request accepted."
		if in.TaskID != "" {
			text += "\nTask ID: " + in.TaskID
		}
		if in.Title != "" {
			text += "\nTitle: " + in.Title
		}
	}

	return StatusReply{
		EditMessageID: in.MessageID,
		Text:          strings.TrimSpace(text),
	}
}
