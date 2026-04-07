package telegram

import (
	"context"
	"time"

	"bili-download/internal/database/models"
)

type UpdatesClient interface {
	GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error)
}

type RuntimeStateStore interface {
	LoadOrCreate(ctx context.Context, botName string) (*models.TelegramRuntimeState, error)
	SaveProgress(ctx context.Context, botName string, lastUpdateID int64, processedUpdateID int64, polledAt time.Time) error
	SaveError(ctx context.Context, botName string, errText string, when time.Time) error
}

type UpdateHandler func(ctx context.Context, update Update) error

type Poller struct {
	botName            string
	client             UpdatesClient
	store              RuntimeStateStore
	pollTimeoutSeconds int
	handle             UpdateHandler
	now                func() time.Time
}

func NewPoller(botName string, client UpdatesClient, store RuntimeStateStore, pollTimeoutSeconds int, handle UpdateHandler) *Poller {
	return &Poller{
		botName:            botName,
		client:             client,
		store:              store,
		pollTimeoutSeconds: pollTimeoutSeconds,
		handle:             handle,
		now:                time.Now,
	}
}

func (p *Poller) Run(ctx context.Context) error {
	state, err := p.store.LoadOrCreate(ctx, p.botName)
	if err != nil {
		return err
	}

	lastUpdateID := state.LastUpdateID
	for {
		if ctx.Err() != nil {
			return nil
		}

		updates, err := p.client.GetUpdates(ctx, lastUpdateID+1, p.pollTimeoutSeconds)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			_ = p.store.SaveError(ctx, p.botName, err.Error(), p.now())
			return err
		}

		if len(updates) == 0 {
			if err := p.store.SaveProgress(ctx, p.botName, lastUpdateID, 0, p.now()); err != nil {
				_ = p.store.SaveError(ctx, p.botName, err.Error(), p.now())
				return err
			}
			continue
		}

		for _, update := range updates {
			if update.UpdateID <= lastUpdateID {
				continue
			}

			if err := p.handle(ctx, update); err != nil {
				_ = p.store.SaveError(ctx, p.botName, err.Error(), p.now())
				return err
			}

			lastUpdateID = update.UpdateID
			if err := p.store.SaveProgress(ctx, p.botName, lastUpdateID, update.UpdateID, p.now()); err != nil {
				_ = p.store.SaveError(ctx, p.botName, err.Error(), p.now())
				return err
			}
		}
	}
}
