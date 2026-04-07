package telegram

import "errors"

var errAccessDenied = errors.New("telegram access denied")

type AccessConfig struct {
	AllowedChatTypes []string
	AllowedChatIDs   []int64
	AllowedUserIDs   []int64
}

func CheckAccess(cfg AccessConfig, chatID int64, userID int64, chatType string) error {
	if !containsString(cfg.AllowedChatTypes, chatType) {
		return errAccessDenied
	}

	if len(cfg.AllowedChatIDs) == 0 && len(cfg.AllowedUserIDs) == 0 {
		return errAccessDenied
	}

	if len(cfg.AllowedChatIDs) > 0 && !containsInt64(cfg.AllowedChatIDs, chatID) {
		return errAccessDenied
	}

	if len(cfg.AllowedUserIDs) > 0 && !containsInt64(cfg.AllowedUserIDs, userID) {
		return errAccessDenied
	}

	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsInt64(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
