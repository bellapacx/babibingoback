package agentbot

import (
	"time"
)

// AgentRequest represents an agent application request
type AgentRequest struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      int64     `gorm:"index"` // Telegram ID
	Username    string
	FirstName   string
	LastName    string
	PhoneNumber string
	Status      string    `gorm:"default:'pending'"` // pending, approved, rejected
	CreatedAt   time.Time
	UpdatedAt   time.Time
}