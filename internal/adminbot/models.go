package adminbot

import (
	"time"
)

// AgentRequest - from agent bot
type AgentRequest struct {
    ID          uint      `gorm:"primaryKey"`
    UserID      int64     `gorm:"index"`
    Username    string
    FirstName   string
    LastName    string
    PhoneNumber string
    Status      string    `gorm:"default:'pending'"` // pending, approved, rejected
    ReviewedBy  *int64
    ReviewedAt  *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// AdminActionLog - audit log
type AdminActionLog struct {
    ID          uint      `gorm:"primaryKey"`
    AdminID     int64     `gorm:"index"`
    AdminName   string
    Action      string    // approve_agent, reject_agent, approve_deposit, etc.
    TargetID    int64     // User ID or Transaction ID
    TargetType  string    // agent, deposit, withdraw, user
    Details     string    `gorm:"type:text"`
    CreatedAt   time.Time
}

// AdminConfig - bot settings
type AdminConfig struct {
    ID               uint      `gorm:"primaryKey"`
	AdminIDs         string    `gorm:"type:text"`
    AutoApprove      bool      `gorm:"default:false"`
    NotifyOnApply    bool      `gorm:"default:true"`
    NotifyOnDeposit  bool      `gorm:"default:true"`
    NotifyOnWithdraw bool      `gorm:"default:true"`
    BotEnabled       bool      `gorm:"default:true"`
    BotsPerTick      int       `gorm:"default:2"`
    MaxBotsPerGame   int       `gorm:"default:50"`
    ReserveInterval  int       `gorm:"default:3"`
    UpdatedAt        time.Time
}

// BotActivityLog - bot activities
type BotActivityLog struct {
    ID          uint      `gorm:"primaryKey"`
    BotID       int64     `gorm:"index"`
    BotName     string
    CardNumber  int
    GameID      string
    Action      string    // reserved, won, played
    CreatedAt   time.Time
}