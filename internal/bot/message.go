package bot

import (
	"context"
	"log"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func (b *Bot) sendText(
	ctx context.Context,
	chatID int64,
	text string,
) {

	_, err := b.api.SendMessage(
		ctx,
		telegoutil.Message(
			telegoutil.ID(chatID),
			text,
		),
	)

	if err != nil {
		log.Printf(
			"failed sending message: %v",
			err,
		)
	}
}


func (b *Bot) sendMarkdown(
	ctx context.Context,
	chatID int64,
	text string,
) {

	msg := telegoutil.Message(
		telegoutil.ID(chatID),
		text,
	)

	msg.ParseMode = "Markdown"

	_, err := b.api.SendMessage(
		ctx,
		msg,
	)

	if err != nil {
		log.Printf(
			"failed sending markdown message: %v",
			err,
		)
	}
}


func (b *Bot) sendMessage(
	ctx context.Context,
	msg *telego.SendMessageParams,
) {

	_, err := b.api.SendMessage(
		ctx,
		msg,
	)

	if err != nil {
		log.Printf(
			"failed sending telegram message: %v",
			err,
		)
	}
}
func (b *Bot) sendMainMenu(
	ctx context.Context,
	chatID int64,
) {

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{
			ID: chatID,
		},

		Text: "Welcome to BabiBingo! 🎱",

		ReplyMarkup: b.mainMenuKeyboard(),
	}


	b.sendMessage(
		ctx,
		&msg,
	)
}