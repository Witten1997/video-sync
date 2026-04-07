package telegram

type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat,omitempty"`
	Text      string `json:"text,omitempty"`
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type ParseResultKind string

const (
	ParseResultKindIgnore ParseResultKind = "ignore"
	ParseResultKindReject ParseResultKind = "reject"
	ParseResultKindSubmit ParseResultKind = "submit"
	ParseResultKindStatus ParseResultKind = "status"
)

type ParseResult struct {
	Kind      ParseResultKind
	URL       string
	TaskID    string
	ReplyText string
}
