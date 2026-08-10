package bot

import (
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func (b *Bot) mainMenuKeyboard() *telego.ReplyKeyboardMarkup {

	return telegoutil.Keyboard(
		telegoutil.KeyboardRow(
			telegoutil.KeyboardButton("🎮 Start play"),
			telegoutil.KeyboardButton("💰 Balance"),
		),

		telegoutil.KeyboardRow(
			telegoutil.KeyboardButton("💳 Deposit"),
			telegoutil.KeyboardButton("🏧 Withdraw"),
		),

		telegoutil.KeyboardRow(
			telegoutil.KeyboardButton("🤝 Agent"),
			telegoutil.KeyboardButton("📨 Invite"),
		),

		telegoutil.KeyboardRow(
			telegoutil.KeyboardButton("🆘 Support"),
		),
	)
}
func (b *Bot) contactKeyboard() *telego.ReplyKeyboardMarkup {

	button := telegoutil.KeyboardButton(
		"📱 Share Phone Number",
	)

	button.RequestContact = true

	return telegoutil.Keyboard(
		telegoutil.KeyboardRow(
			button,
		),
	)
}
func (b *Bot) playKeyboard() *telego.InlineKeyboardMarkup {

	keyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{
					Text: "🎮 Open BabiBingo",
					WebApp: &telego.WebAppInfo{
						URL: b.webAppURL,
					},
				},
			},
		},
	}

	fmt.Printf("%+v\n", keyboard)

	return keyboard
}