package telegram

import (
	"regexp"
	"strings"
)

var urlPattern = regexp.MustCompile(`https?://[^\s]+`)

func ExtractURLs(text string, maxURLs int) []string {
	rawURLs := extractURLsRaw(text)
	if maxURLs > 0 && len(rawURLs) > maxURLs {
		return rawURLs[:maxURLs]
	}
	return rawURLs
}

func ParseMessage(text string, maxURLs int) ParseResult {
	return ParseMessageForChat(text, maxURLs, "private", "")
}

func ParseMessageForChat(text string, maxURLs int, chatType string, botUsername string) ParseResult {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ParseResult{Kind: ParseResultKindIgnore}
	}

	normalizedBotUsername := normalizeBotUsername(botUsername)
	groupScoped := isGroupChatType(chatType)

	if command, addressed := parseCommand(trimmed, "download", normalizedBotUsername, groupScoped); addressed {
		commandBody := strings.TrimSpace(commandArgument(command))
		if commandBody == "" {
			return ParseResult{
				Kind:      ParseResultKindReject,
				ReplyText: "用法：/download <url>",
			}
		}
		return parseURLText(commandBody, maxURLs, true)
	}

	if command, addressed := parseCommand(trimmed, "status", normalizedBotUsername, groupScoped); addressed {
		return ParseResult{
			Kind:   ParseResultKindStatus,
			TaskID: strings.TrimSpace(commandArgument(command)),
		}
	}

	if _, addressed := parseCommand(trimmed, "help", normalizedBotUsername, groupScoped); addressed {
		return ParseResult{
			Kind:      ParseResultKindHelp,
			ReplyText: buildHelpText(),
		}
	}

	if groupScoped {
		mentionedText, addressed := extractLeadingMentionPayload(trimmed, normalizedBotUsername)
		if !addressed {
			return ParseResult{Kind: ParseResultKindIgnore}
		}
		return ParseMessage(mentionedText, maxURLs)
	}

	return parseDirectURLMessage(trimmed, maxURLs)
}

func parseURLText(text string, maxURLs int, command bool) ParseResult {
	urls := extractURLsRaw(text)
	if len(urls) == 0 {
		if command {
			return ParseResult{
				Kind:      ParseResultKindReject,
				ReplyText: "用法：/download <url>",
			}
		}
		return ParseResult{Kind: ParseResultKindIgnore}
	}

	if maxURLs <= 0 {
		maxURLs = 1
	}

	if len(urls) > maxURLs {
		return ParseResult{
			Kind:      ParseResultKindReject,
			ReplyText: "每条消息只支持一个 URL。",
		}
	}

	return ParseResult{
		Kind: ParseResultKindSubmit,
		URL:  urls[0],
	}
}

func parseDirectURLMessage(text string, maxURLs int) ParseResult {
	result := parseURLText(text, maxURLs, false)
	if result.Kind != ParseResultKindSubmit {
		return result
	}
	if strings.TrimSpace(result.URL) != text {
		return ParseResult{Kind: ParseResultKindIgnore}
	}
	return result
}

func extractURLsRaw(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		cleaned := cleanURL(match)
		if cleaned == "" {
			continue
		}
		urls = append(urls, cleaned)
	}
	return urls
}

func cleanURL(raw string) string {
	return strings.Trim(raw, "<>\"'()[]{}.,;!?")
}

func isDownloadCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}

	return fields[0] == "/download"
}

func isStatusCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}

	return fields[0] == "/status"
}

func buildHelpText() string {
	return strings.Join([]string{
		"可用命令：",
		"/download <url> - 提交下载链接",
		"/status [任务ID] - 查询最近请求或指定任务状态",
		"/help - 查看帮助",
		"",
		"使用说明：",
		"1. 私聊可直接发送单个 URL。",
		"2. 群聊请使用 /download@botname、/status@botname、/help@botname，或以 @botname 开头。",
	}, "\n")
}

func commandArgument(text string) string {
	fields := strings.Fields(text)
	if len(fields) <= 1 {
		return ""
	}
	return strings.Join(fields[1:], " ")
}

func parseCommand(text string, commandName string, botUsername string, requireBotMention bool) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", false
	}

	commandToken := fields[0]
	baseCommand := "/" + commandName
	if commandToken == baseCommand {
		if requireBotMention {
			return "", false
		}
		return text, true
	}

	if botUsername == "" {
		return "", false
	}

	expectedMentionCommand := baseCommand + "@" + botUsername
	if strings.EqualFold(commandToken, expectedMentionCommand) {
		return strings.Join(append([]string{baseCommand}, fields[1:]...), " "), true
	}

	return "", false
}

func extractLeadingMentionPayload(text string, botUsername string) (string, bool) {
	if botUsername == "" {
		return "", false
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", false
	}

	if !strings.EqualFold(fields[0], "@"+botUsername) {
		return "", false
	}

	if len(fields) == 1 {
		return "", true
	}

	return strings.Join(fields[1:], " "), true
}

func normalizeBotUsername(botUsername string) string {
	botUsername = strings.TrimSpace(botUsername)
	botUsername = strings.TrimPrefix(botUsername, "@")
	return strings.ToLower(botUsername)
}

func isGroupChatType(chatType string) bool {
	switch strings.ToLower(strings.TrimSpace(chatType)) {
	case "group", "supergroup":
		return true
	default:
		return false
	}
}
