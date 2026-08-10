package bot

import (
	"context"
	"log"

	"github.com/mymmrac/telego"
)

func (b *Bot) Start(ctx context.Context) error {

	updates, err := b.api.UpdatesViaLongPolling(
		ctx,
		&telego.GetUpdatesParams{
			Timeout: 60,
		},
	)
	if err != nil {
		return err
	}

	log.Println("Telegram bot is listening for updates")

	for update := range updates {
		go b.handleUpdate(ctx, update)
	}

	return nil
}


func (b *Bot) handleUpdate(
	ctx context.Context,
	update telego.Update,
) {

	switch {

	case update.Message != nil:
		b.handleMessage(
			ctx,
			update.Message,
		)

	case update.CallbackQuery != nil:
		b.handleCallback(
			ctx,
			update.CallbackQuery,
		)

	}
}