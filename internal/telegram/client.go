package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/utils"
)

type BotAPI interface {
	GetMe(ctx context.Context) (*User, error)
	GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error)
	SendMessage(ctx context.Context, chatID int64, text string, replyToMessageID int64) (*Message, error)
	EditMessageText(ctx context.Context, chatID int64, messageID int64, text string) (*Message, error)
	SetWebhook(ctx context.Context, webhookURL string, secretToken string) error
	DeleteWebhook(ctx context.Context, dropPendingUpdates bool) error
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(token string, pollTimeoutSeconds int, proxyCfg config.ProxyConfig) *Client {
	timeout := time.Duration(pollTimeoutSeconds+10) * time.Second
	if timeout <= 0 {
		timeout = 40 * time.Second
	}

	return &Client{
		baseURL:    "https://api.telegram.org",
		token:      token,
		httpClient: utils.NewHTTPClient(proxyCfg, timeout, 20, 10),
	}
}

func (c *Client) GetMe(ctx context.Context) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("getMe"), nil)
	if err != nil {
		return nil, err
	}

	var resp apiResponse[User]
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram getMe failed: %s", resp.Description)
	}

	return &resp.Result, nil
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	query := url.Values{}
	query.Set("offset", fmt.Sprintf("%d", offset))
	query.Set("timeout", fmt.Sprintf("%d", timeoutSeconds))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("getUpdates")+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var resp apiResponse[[]Update]
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram getUpdates failed: %s", resp.Description)
	}

	return resp.Result, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, replyToMessageID int64) (*Message, error) {
	form := url.Values{}
	form.Set("chat_id", fmt.Sprintf("%d", chatID))
	form.Set("text", text)
	if replyToMessageID > 0 {
		form.Set("reply_to_message_id", fmt.Sprintf("%d", replyToMessageID))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("sendMessage"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp apiResponse[Message]
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram sendMessage failed: %s", resp.Description)
	}

	return &resp.Result, nil
}

func (c *Client) EditMessageText(ctx context.Context, chatID int64, messageID int64, text string) (*Message, error) {
	form := url.Values{}
	form.Set("chat_id", fmt.Sprintf("%d", chatID))
	form.Set("message_id", fmt.Sprintf("%d", messageID))
	form.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("editMessageText"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp apiResponse[Message]
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram editMessageText failed: %s", resp.Description)
	}

	return &resp.Result, nil
}

func (c *Client) SetWebhook(ctx context.Context, webhookURL string, secretToken string) error {
	form := url.Values{}
	form.Set("url", webhookURL)
	if strings.TrimSpace(secretToken) != "" {
		form.Set("secret_token", secretToken)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("setWebhook"), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp apiResponse[bool]
	if err := c.doJSON(req, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram setWebhook failed: %s", resp.Description)
	}

	return nil
}

func (c *Client) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) error {
	form := url.Values{}
	if dropPendingUpdates {
		form.Set("drop_pending_updates", "true")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("deleteWebhook"), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp apiResponse[bool]
	if err := c.doJSON(req, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram deleteWebhook failed: %s", resp.Description)
	}

	return nil
}

func (c *Client) endpoint(method string) string {
	return strings.TrimRight(c.baseURL, "/") + "/bot" + c.token + "/" + method
}

func (c *Client) doJSON(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode telegram response: %w", err)
	}

	return nil
}
