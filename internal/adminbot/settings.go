package adminbot

import (
	"context"
	"fmt"
	"strconv"
)

func (b *Bot) handleSettings(ctx context.Context, chatID int64, args []string) {
    if len(args) == 0 {
        b.showSettings(ctx, chatID)
        return
    }

    switch args[0] {
    case "admins":
        b.showAdmins(ctx, chatID)
    case "addadmin":
        if len(args) > 1 {
            id, _ := strconv.ParseInt(args[1], 10, 64)
            b.addAdmin(ctx, chatID, id)
        }
    case "removeadmin":
        if len(args) > 1 {
            id, _ := strconv.ParseInt(args[1], 10, 64)
            b.removeAdmin(ctx, chatID, id)
        }
    case "autoapprove":
        b.toggleAutoApprove(ctx, chatID)
    case "notifications":
        b.toggleNotifications(ctx, chatID)
    default:
        b.sendText(ctx, chatID, "❌ Usage: /settings [admins|addadmin <id>|removeadmin <id>|autoapprove|notifications]")
    }
}

func (b *Bot) showSettings(ctx context.Context, chatID int64) {
    var config AdminConfig
    b.db.FirstOrCreate(&config, AdminConfig{
        AutoApprove:      false,
        NotifyOnApply:    true,
        NotifyOnDeposit:  true,
        NotifyOnWithdraw: true,
        BotEnabled:       true,
        BotsPerTick:      2,
        MaxBotsPerGame:   50,
        ReserveInterval:  3,
    })

    adminList := ""
    for _, id := range b.admins {
        adminList += fmt.Sprintf("• `%d`\n", id)
    }

    b.sendMarkdown(
        ctx,
        chatID,
        fmt.Sprintf(
            "⚙️ *Bot Settings*\n\n"+
                "👥 *Admins:*\n%s\n"+
                "📋 *Agent Settings:*\n"+
                "• Auto-approve: %v\n"+
                "• Notify on apply: %v\n\n"+
                "💳 *Financial:*\n"+
                "• Notify on deposit: %v\n"+
                "• Notify on withdraw: %v\n\n"+
                "🤖 *Bot Settings:*\n"+
                "• Enabled: %v\n"+
                "• Bots per tick: %d\n"+
                "• Max bots per game: %d\n"+
                "• Reserve interval: %ds",
            adminList,
            config.AutoApprove,
            config.NotifyOnApply,
            config.NotifyOnDeposit,
            config.NotifyOnWithdraw,
            config.BotEnabled,
            config.BotsPerTick,
            config.MaxBotsPerGame,
            config.ReserveInterval,
        ),
    )
}
// settings.go - Add these missing functions

func (b *Bot) showAdmins(ctx context.Context, chatID int64) {
    adminList := ""
    for _, id := range b.admins {
        adminList += fmt.Sprintf("• `%d`\n", id)
    }

    b.sendMarkdown(
        ctx,
        chatID,
        fmt.Sprintf(
            "👥 *Admin List*\n\n%s\n\n"+
                "Total: %d admins",
            adminList,
            len(b.admins),
        ),
    )
}

func (b *Bot) addAdmin(ctx context.Context, chatID int64, newAdminID int64) {
    // Check if already admin
    for _, id := range b.admins {
        if id == newAdminID {
            b.sendText(ctx, chatID, fmt.Sprintf("❌ User %d is already an admin.", newAdminID))
            return
        }
    }

    // Add to list (in memory - will reset on restart)
    b.admins = append(b.admins, newAdminID)

    // TODO: If using database, save to DB here

    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "✅ *Admin Added*\n\nUser `%d` is now an administrator.\n\n"+
            "⚠️ This change will persist until bot restart (or until saved to database).",
        newAdminID,
    ))

    // Log action
    b.logAdminAction(ctx, chatID, "add_admin", newAdminID, "admin", fmt.Sprintf("Added admin %d", newAdminID))
}

func (b *Bot) removeAdmin(ctx context.Context, chatID int64, adminID int64) {
    // Find and remove
    found := false
    newAdmins := []int64{}
    for _, id := range b.admins {
        if id == adminID {
            found = true
            continue
        }
        newAdmins = append(newAdmins, id)
    }

    if !found {
        b.sendText(ctx, chatID, fmt.Sprintf("❌ Admin %d not found.", adminID))
        return
    }

    b.admins = newAdmins

    // TODO: If using database, remove from DB here

    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "❌ *Admin Removed*\n\nUser `%d` is no longer an administrator.\n\n"+
            "⚠️ This change will persist until bot restart (or until saved to database).",
        adminID,
    ))

    // Log action
    b.logAdminAction(ctx, chatID, "remove_admin", adminID, "admin", fmt.Sprintf("Removed admin %d", adminID))
}

func (b *Bot) toggleAutoApprove(ctx context.Context, chatID int64) {
    var config AdminConfig
    b.db.FirstOrCreate(&config, AdminConfig{
        AutoApprove:      false,
        NotifyOnApply:    true,
        NotifyOnDeposit:  true,
        NotifyOnWithdraw: true,
        BotEnabled:       true,
        BotsPerTick:      2,
        MaxBotsPerGame:   50,
        ReserveInterval:  3,
    })

    // Toggle
    config.AutoApprove = !config.AutoApprove
    b.db.Save(&config)

    status := "✅ Enabled"
    if !config.AutoApprove {
        status = "❌ Disabled"
    }

    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "⚙️ *Auto-Approve Updated*\n\n"+
            "Auto-approve for agents is now: %s\n\n"+
            "• Enabled: Applications are auto-approved\n"+
            "• Disabled: Admin must manually approve",
        status,
    ))

    b.logAdminAction(ctx, chatID, "toggle_auto_approve", 0, "settings", fmt.Sprintf("Set auto-approve to %v", config.AutoApprove))
}

func (b *Bot) toggleNotifications(ctx context.Context, chatID int64) {
    var config AdminConfig
    b.db.FirstOrCreate(&config, AdminConfig{
        AutoApprove:      false,
        NotifyOnApply:    true,
        NotifyOnDeposit:  true,
        NotifyOnWithdraw: true,
        BotEnabled:       true,
        BotsPerTick:      2,
        MaxBotsPerGame:   50,
        ReserveInterval:  3,
    })

    // Toggle all notifications
    config.NotifyOnApply = !config.NotifyOnApply
    config.NotifyOnDeposit = !config.NotifyOnDeposit
    config.NotifyOnWithdraw = !config.NotifyOnWithdraw
    b.db.Save(&config)

    status := "✅ Enabled"
    if !config.NotifyOnApply {
        status = "❌ Disabled"
    }

    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "🔔 *Notifications Updated*\n\n"+
            "Notifications are now: %s\n\n"+
            "• Agent applications: %v\n"+
            "• Deposits: %v\n"+
            "• Withdrawals: %v",
        status,
        config.NotifyOnApply,
        config.NotifyOnDeposit,
        config.NotifyOnWithdraw,
    ))

    b.logAdminAction(ctx, chatID, "toggle_notifications", 0, "settings", fmt.Sprintf("Set notifications to %v", config.NotifyOnApply))
}