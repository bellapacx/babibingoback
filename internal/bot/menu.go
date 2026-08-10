package bot

import (
	"context"

	"github.com/mymmrac/telego"
)

func (b *Bot) setupCommands(ctx context.Context) error {
	commands := []telego.BotCommand{
		{
			Command:     "start",
			Description: "Start BabiBingo",
		},
		{
			Command:     "play",
			Description: "Open the game",
		},
		{
			Command:     "balance",
			Description: "View your balance",
		},
		{
			Command:     "deposit",
			Description: "Deposit funds",
		},
		{
			Command:     "withdraw",
			Description: "Withdraw winnings",
		},
		{
			Command:     "agent",
			Description: "Agent dashboard",
		},
		{
			Command:     "invite",
			Description: "Invite friends",
		},
		{
			Command:     "support",
			Description: "Contact support",
		},
	}

	err := b.api.SetMyCommands(
		ctx,
		&telego.SetMyCommandsParams{
			Commands: commands,
		},
	)

	return err
}