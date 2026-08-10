package adminbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"babibingo/internal/models"

	"github.com/google/uuid"
	"github.com/mymmrac/telego"
	"gorm.io/gorm"
)

// handleGames - Main game monitoring handler with interactive menu
func (b *Bot) handleGames(ctx context.Context, chatID int64, args []string) {
	if len(args) == 0 {
		b.showGameMenu(ctx, chatID)
		return
	}

	switch args[0] {
	case "active":
		b.showActiveGames(ctx, chatID)
	case "current":
		b.showCurrentGame(ctx, chatID)
	case "history":
		b.showGameHistory(ctx, chatID)
	case "stats":
		b.showGameStats(ctx, chatID)
	case "end":
		if len(args) > 1 {
			gameID := args[1]
			b.forceEndGame(ctx, chatID, gameID)
		} else {
			b.sendText(ctx, chatID, "❌ Usage: /games end <game_id>")
		}
	case "pool":
		b.showCurrentPool(ctx, chatID)
	default:
		b.sendText(ctx, chatID, "❌ Usage: /games [active|current|history|stats|end <id>|pool]")
	}
}

// ✅ showGameMenu - Interactive game menu
func (b *Bot) showGameMenu(ctx context.Context, chatID int64) {
	// Get counts
	var activeGames int64
	b.db.Model(&models.Game{}).Where("status IN (?)", []string{"waiting", "calling"}).Count(&activeGames)

	var totalGames int64
	b.db.Model(&models.Game{}).Count(&totalGames)

	var todayGames int64
	today := time.Now().Truncate(24 * time.Hour)
	b.db.Model(&models.Game{}).Where("created_at >= ?", today).Count(&todayGames)

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf(
			"🎱 *Game Monitoring*\n\n"+
				"📊 *Overview:*\n"+
				"• 🟢 Active Games: %d\n"+
				"• 📋 Total Games: %d\n"+
				"• 📅 Today: %d\n\n"+
				"Select an option below:",
			activeGames,
			totalGames,
			todayGames,
		),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         fmt.Sprintf("🟢 Active (%d)", activeGames),
						CallbackData: "games_active",
					},
					{
						Text:         "🎯 Current",
						CallbackData: "games_current",
					},
				},
				{
					{
						Text:         "📋 History",
						CallbackData: "games_history",
					},
					{
						Text:         "📊 Stats",
						CallbackData: "games_stats",
					},
				},
				{
					{
						Text:         "💰 Pool",
						CallbackData: "games_pool",
					},
				},
				{
					{
						Text:         "🔙 Back",
						CallbackData: "back_to_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ showActiveGames - Consolidated active games list
func (b *Bot) showActiveGames(ctx context.Context, chatID int64) {
	var games []models.Game
	b.db.Where("status IN (?)", []string{"waiting", "calling"}).
		Order("created_at DESC").
		Find(&games)

	if len(games) == 0 {
		msg := telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   "🎱 No active games.",
			ReplyMarkup: &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{
							Text:         "🔙 Back",
							CallbackData: "games_menu",
						},
					},
				},
			},
		}
		b.sendMessage(ctx, &msg)
		return
	}

	// ✅ Build a single consolidated message
	var gameList strings.Builder
	gameList.WriteString(fmt.Sprintf("🎱 *Active Games*\n\nTotal: %d\n\n", len(games)))

	for i, game := range games {
		var playerCount int64
		b.db.Model(&models.GamePlayer{}).Where("game_id = ?", game.ID).Count(&playerCount)

		var cardCount int64
		b.db.Model(&models.Card{}).Where("game_id = ?", game.ID).Count(&cardCount)

		statusEmoji := "🟡"
		statusText := "Waiting"
		if game.Status == "calling" {
			statusEmoji = "🔵"
			statusText = "Calling"
		}

		gameList.WriteString(fmt.Sprintf(
			"%d. `%s` %s %s | 👥%d 🃏%d 💰%.0f\n",
			i+1,
			game.ID.String()[:8],
			statusEmoji,
			statusText,
			playerCount,
			cardCount,
			game.TotalPool,
		))
	}

	// ✅ Add note about using /games current for details
	gameList.WriteString("\n💡 Use /games current to see full game details")

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   gameList.String(),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "🎯 Current Game",
						CallbackData: "games_current",
					},
				},
				{
					{
						Text:         "📊 Stats",
						CallbackData: "games_stats",
					},
					{
						Text:         "🔙 Back",
						CallbackData: "games_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ showCurrentGame - Show current game details (improved)
func (b *Bot) showCurrentGame(ctx context.Context, chatID int64) {
	var game models.Game
	err := b.db.Where("status IN (?)", []string{"waiting", "calling"}).
		Order("created_at DESC").
		First(&game).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			msg := telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   "🎱 No active game.",
				ReplyMarkup: &telego.InlineKeyboardMarkup{
					InlineKeyboard: [][]telego.InlineKeyboardButton{
						{
							{
								Text:         "🔄 Refresh",
								CallbackData: "games_current",
							},
						},
						{
							{
								Text:         "🔙 Back",
								CallbackData: "games_menu",
							},
						},
					},
				},
			}
			b.sendMessage(ctx, &msg)
			return
		}
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Error: %v", err))
		return
	}

	// Get counts
	var playerCount int64
	b.db.Model(&models.GamePlayer{}).Where("game_id = ?", game.ID).Count(&playerCount)

	var cardCount int64
	b.db.Model(&models.Card{}).Where("game_id = ?", game.ID).Count(&cardCount)

	calledCount := len(game.CalledNumbers)

	// Get top players
	var topPlayers []struct {
		UserID     int64
		CardsCount int
		Username   string
	}
	b.db.Table("game_players").
		Select("game_players.user_id, game_players.cards_count, users.username").
		Joins("LEFT JOIN users ON users.id = game_players.user_id").
		Where("game_players.game_id = ?", game.ID).
		Order("game_players.cards_count DESC").
		Limit(5).
		Scan(&topPlayers)

	topPlayersText := ""
	if len(topPlayers) > 0 {
		for i, player := range topPlayers {
			username := player.Username
			if username == "" {
				username = fmt.Sprintf("User %d", player.UserID)
			}
			topPlayersText += fmt.Sprintf("%d. @%s - %d cards\n", i+1, username, player.CardsCount)
		}
	} else {
		topPlayersText = "No players yet"
	}

	statusEmoji := "🟡"
	statusText := "Waiting"
	if game.Status == "calling" {
		statusEmoji = "🔵"
		statusText = "Calling"
	}

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf(
			"🎱 *Current Game*\n\n"+
				"🆔 ID: `%s`\n"+
				"📊 Status: %s %s\n"+
				"👥 Players: %d\n"+
				"🃏 Cards: %d/400\n"+
				"💰 Pool: %.2f ETB\n"+
				"🔢 Called: %d/75\n"+
				"📅 Started: %s\n\n"+
				"🏆 *Top Players:*\n%s",
			game.ID.String()[:8],
			statusEmoji,
			statusText,
			playerCount,
			cardCount,
			game.TotalPool,
			calledCount,
			game.CreatedAt.Format("Jan 2, 15:04"),
			topPlayersText,
		),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "🔄 Refresh",
						CallbackData: "games_current",
					},
					{
						Text:         "💰 Pool",
						CallbackData: "games_pool",
					},
				},
				{
					{
						Text:         "📊 Stats",
						CallbackData: "games_stats",
					},
					{
						Text:         "🔙 Back",
						CallbackData: "games_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ showGameHistory - Consolidated history
func (b *Bot) showGameHistory(ctx context.Context, chatID int64) {
	var games []models.Game
	b.db.Where("status = ?", "finished").
		Order("ended_at DESC").
		Limit(10).
		Find(&games)

	if len(games) == 0 {
		msg := telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   "📋 No game history found.",
			ReplyMarkup: &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{
							Text:         "🔙 Back",
							CallbackData: "games_menu",
						},
					},
				},
			},
		}
		b.sendMessage(ctx, &msg)
		return
	}

	// ✅ Build consolidated history
	var history strings.Builder
	history.WriteString("📋 *Game History (Last 10)*\n\n")

	for i, game := range games {
		var winnerName string
		if game.WinnerUserID != nil {
			var user models.User
			if err := b.db.First(&user, *game.WinnerUserID).Error; err == nil {
				winnerName = "@" + user.Username
			} else {
				winnerName = fmt.Sprintf("User %d", *game.WinnerUserID)
			}
		} else {
			winnerName = "No winner"
		}

		history.WriteString(fmt.Sprintf(
			"%d. %s | 💰%.0f | 🃏%d | 🏆%s\n",
			i+1,
			game.EndedAt.Format("Jan 2, 15:04"),
			game.TotalPool,
			len(game.CalledNumbers),
			winnerName,
		))
	}

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   history.String(),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "🔄 Refresh",
						CallbackData: "games_history",
					},
				},
				{
					{
						Text:         "📊 Stats",
						CallbackData: "games_stats",
					},
					{
						Text:         "🔙 Back",
						CallbackData: "games_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ showGameStats - Show game statistics with buttons
func (b *Bot) showGameStats(ctx context.Context, chatID int64) {
	var totalGames int64
	var totalPool float64
	var avgPool float64
	var totalPlayers int64
	var totalCards int64

	b.db.Model(&models.Game{}).Count(&totalGames)
	b.db.Model(&models.Game{}).Select("COALESCE(SUM(total_pool), 0)").Scan(&totalPool)

	if totalGames > 0 {
		avgPool = totalPool / float64(totalGames)
	}

	b.db.Model(&models.GamePlayer{}).Count(&totalPlayers)
	b.db.Model(&models.Card{}).Count(&totalCards)

	today := time.Now().Truncate(24 * time.Hour)
	var todayGames int64
	b.db.Model(&models.Game{}).Where("created_at >= ?", today).Count(&todayGames)

	var activeGames int64
	b.db.Model(&models.Game{}).Where("status IN (?)", []string{"waiting", "calling"}).Count(&activeGames)

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf(
			"📊 *Game Statistics*\n\n"+
				"🎱 *Games:*\n"+
				"• Total Games: %d\n"+
				"• Active Games: %d\n"+
				"• Today's Games: %d\n\n"+
				"💰 *Pool:*\n"+
				"• Total Pool: %.2f ETB\n"+
				"• Average Pool: %.2f ETB\n\n"+
				"👥 *Players:*\n"+
				"• Total Players: %d\n"+
				"• Total Cards: %d\n"+
				"• Avg Cards/Player: %.1f",
			totalGames,
			activeGames,
			todayGames,
			totalPool,
			avgPool,
			totalPlayers,
			totalCards,
			float64(totalCards)/float64(totalPlayers),
		),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "🎯 Active Games",
						CallbackData: "games_active",
					},
					{
						Text:         "💰 Pool",
						CallbackData: "games_pool",
					},
				},
				{
					{
						Text:         "🔄 Refresh",
						CallbackData: "games_stats",
					},
					{
						Text:         "🔙 Back",
						CallbackData: "games_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ showCurrentPool - Show pool details with buttons
func (b *Bot) showCurrentPool(ctx context.Context, chatID int64) {
	var game models.Game
	err := b.db.Where("status IN (?)", []string{"waiting", "calling"}).
		Order("created_at DESC").
		First(&game).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			msg := telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   "🎱 No active game.",
				ReplyMarkup: &telego.InlineKeyboardMarkup{
					InlineKeyboard: [][]telego.InlineKeyboardButton{
						{
							{
								Text:         "🔙 Back",
								CallbackData: "games_menu",
							},
						},
					},
				},
			}
			b.sendMessage(ctx, &msg)
			return
		}
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Error: %v", err))
		return
	}

	var playerCount int64
	b.db.Model(&models.GamePlayer{}).Where("game_id = ?", game.ID).Count(&playerCount)

	var cardCount int64
	b.db.Model(&models.Card{}).Where("game_id = ?", game.ID).Count(&cardCount)

	houseCut := game.TotalPool * 0.10
	prizePool := game.TotalPool - houseCut

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf(
			"💰 *Current Pool Details*\n\n"+
				"🎱 Game ID: `%s`\n"+
				"📊 Status: %s\n"+
				"👥 Players: %d\n"+
				"🃏 Cards: %d\n\n"+
				"💰 *Pool Breakdown:*\n"+
				"• Gross Pool: %.2f ETB\n"+
				"• House Cut (10%%): %.2f ETB\n"+
				"• Prize Pool: %.2f ETB\n\n"+
				"📅 Started: %s",
			game.ID.String()[:8],
			game.Status,
			playerCount,
			cardCount,
			game.TotalPool,
			houseCut,
			prizePool,
			game.CreatedAt.Format("Jan 2, 15:04"),
		),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "🎯 Current Game",
						CallbackData: "games_current",
					},
				},
				{
					{
						Text:         "🔄 Refresh",
						CallbackData: "games_pool",
					},
					{
						Text:         "🔙 Back",
						CallbackData: "games_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ forceEndGame - Force end a game with confirmation
func (b *Bot) forceEndGame(ctx context.Context, chatID int64, gameID string) {
	id, err := uuid.Parse(gameID)
	if err != nil {
		b.sendText(ctx, chatID, "❌ Invalid game ID format. Use UUID format.")
		return
	}

	var game models.Game
	if err := b.db.Where("id = ?", id).First(&game).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			b.sendText(ctx, chatID, "❌ Game not found.")
		} else {
			b.sendText(ctx, chatID, fmt.Sprintf("❌ Error: %v", err))
		}
		return
	}

	if game.Status == "finished" {
		b.sendText(ctx, chatID, "⚠️ This game is already finished.")
		return
	}

	// ✅ Show confirmation dialog
	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf(
			"⚠️ *Force End Game*\n\n"+
				"Are you sure you want to force end this game?\n\n"+
				"🆔 Game ID: `%s`\n"+
				"💰 Current Pool: %.2f ETB\n"+
				"👥 Players: %d\n\n"+
				"This action cannot be undone!",
			game.ID.String()[:8],
			game.TotalPool,
			func() int64 {
				var count int64
				b.db.Model(&models.GamePlayer{}).Where("game_id = ?", game.ID).Count(&count)
				return count
			}(),
		),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "✅ Yes, Force End",
						CallbackData: fmt.Sprintf("games_end_confirm_%s", game.ID.String()),
					},
					{
						Text:         "❌ Cancel",
						CallbackData: "games_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ forceEndGameConfirm - Execute force end after confirmation
func (b *Bot) forceEndGameConfirm(ctx context.Context, chatID int64, gameID string) {
	id, err := uuid.Parse(gameID)
	if err != nil {
		b.sendText(ctx, chatID, "❌ Invalid game ID.")
		return
	}

	var game models.Game
	if err := b.db.Where("id = ?", id).First(&game).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Game not found.")
		return
	}

	now := time.Now()
	game.Status = "finished"
	game.EndedAt = &now

	if err := b.db.Save(&game).Error; err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Failed to end game: %v", err))
		return
	}

	b.logAdminAction(ctx, chatID, "force_end_game", 0, "game",
		fmt.Sprintf("Force ended game %s", gameID))

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *Game Ended*\n\n"+
			"🆔 Game ID: `%s`\n"+
			"💰 Final Pool: %.2f ETB\n"+
			"📅 Ended at: %s\n\n"+
			"⚠️ This game was force-ended by an administrator.",
		game.ID.String()[:8],
		game.TotalPool,
		now.Format("Jan 2, 2006 15:04"),
	))
}