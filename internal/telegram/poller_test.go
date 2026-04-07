package telegram

import (
	"context"
	"testing"
	"time"

	"bili-download/internal/database/models"
)

type fakePollerClient struct {
	batches [][]Update
	offsets []int64
	call    int
	cancel  context.CancelFunc
}

func (c *fakePollerClient) GetUpdates(_ context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	c.offsets = append(c.offsets, offset)
	if timeoutSeconds <= 0 {
		return nil, nil
	}

	if c.call >= len(c.batches) {
		if c.cancel != nil {
			c.cancel()
		}
		return []Update{}, nil
	}

	batch := c.batches[c.call]
	c.call++
	if c.call >= len(c.batches) && c.cancel != nil {
		c.cancel()
	}
	return batch, nil
}

type fakeRuntimeStateStore struct {
	state       *models.TelegramRuntimeState
	savedIDs    []int64
	pollTimes   []time.Time
	errorTexts  []string
	errorTimes  []time.Time
	loadBotName string
}

func (s *fakeRuntimeStateStore) LoadOrCreate(_ context.Context, botName string) (*models.TelegramRuntimeState, error) {
	s.loadBotName = botName
	if s.state == nil {
		s.state = &models.TelegramRuntimeState{BotName: botName}
	}
	return s.state, nil
}

func (s *fakeRuntimeStateStore) SaveProgress(_ context.Context, botName string, lastUpdateID int64, processedUpdateID int64, polledAt time.Time) error {
	s.loadBotName = botName
	s.state.LastUpdateID = lastUpdateID
	s.state.WebhookRecentUpdateIDs = appendWebhookRecentUpdateID(s.state.WebhookRecentUpdateIDs, processedUpdateID)
	s.savedIDs = append(s.savedIDs, lastUpdateID)
	s.pollTimes = append(s.pollTimes, polledAt)
	return nil
}

func (s *fakeRuntimeStateStore) SaveError(_ context.Context, botName string, errText string, when time.Time) error {
	s.loadBotName = botName
	s.errorTexts = append(s.errorTexts, errText)
	s.errorTimes = append(s.errorTimes, when)
	return nil
}

func TestPollerStartsFromPersistedOffset(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	client := &fakePollerClient{
		batches: [][]Update{{}},
		cancel:  cancel,
	}
	store := &fakeRuntimeStateStore{
		state: &models.TelegramRuntimeState{
			BotName:      "bot",
			LastUpdateID: 41,
		},
	}

	poller := NewPoller("bot", client, store, 30, func(context.Context, Update) error { return nil })
	poller.now = func() time.Time { return time.Unix(1700000000, 0) }

	if err := poller.Run(ctx); err != nil {
		t.Fatalf("expected poller to exit cleanly, got %v", err)
	}

	if len(client.offsets) == 0 {
		t.Fatal("expected at least one getUpdates call")
	}
	if client.offsets[0] != 42 {
		t.Fatalf("expected first offset 42, got %d", client.offsets[0])
	}
}

func TestPollerSkipsDuplicateUpdatesAndPersistsProgress(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	client := &fakePollerClient{
		batches: [][]Update{
			{
				{UpdateID: 100},
				{UpdateID: 101},
				{UpdateID: 101},
				{UpdateID: 102},
			},
		},
		cancel: cancel,
	}
	store := &fakeRuntimeStateStore{
		state: &models.TelegramRuntimeState{
			BotName:      "bot",
			LastUpdateID: 100,
		},
	}

	var handled []int64
	poller := NewPoller("bot", client, store, 30, func(_ context.Context, update Update) error {
		handled = append(handled, update.UpdateID)
		return nil
	})
	poller.now = func() time.Time { return time.Unix(1700000100, 0) }

	if err := poller.Run(ctx); err != nil {
		t.Fatalf("expected poller to exit cleanly, got %v", err)
	}

	if len(handled) != 2 {
		t.Fatalf("expected exactly 2 handled updates, got %d (%v)", len(handled), handled)
	}
	if handled[0] != 101 || handled[1] != 102 {
		t.Fatalf("expected handled updates [101 102], got %v", handled)
	}

	if len(store.savedIDs) != 2 {
		t.Fatalf("expected 2 progress saves, got %d (%v)", len(store.savedIDs), store.savedIDs)
	}
	if store.savedIDs[0] != 101 || store.savedIDs[1] != 102 {
		t.Fatalf("expected saved ids [101 102], got %v", store.savedIDs)
	}
}
