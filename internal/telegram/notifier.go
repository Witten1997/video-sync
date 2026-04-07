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
		text = "下载完成。"
		if in.Title != "" {
			text += "\n标题：" + in.Title
		}
		if in.RecordID > 0 {
			text += "\n记录 ID：" + fmt.Sprint(in.RecordID)
		}
	case TelegramRequestStatusFailed:
		text = "下载失败。"
		if in.ErrorMessage != "" {
			text += "\n原因：" + in.ErrorMessage
		}
	default:
		text = "已受理请求。"
		if in.TaskID != "" {
			text += "\n任务 ID：" + in.TaskID
		}
		if in.Title != "" {
			text += "\n标题：" + in.Title
		}
	}

	return StatusReply{
		EditMessageID: in.MessageID,
		Text:          strings.TrimSpace(text),
	}
}
