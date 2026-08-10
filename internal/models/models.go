package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User represents a Telegram user
type User struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	TelegramID    int64     `gorm:"uniqueIndex" json:"telegram_id"`
	PhoneNumber   string    `json:"phone_number"`
	Username      string    `json:"username"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Balance       float64   `gorm:"default:0" json:"balance"`
	ReferralCode  string    `gorm:"uniqueIndex" json:"referral_code"`
	ReferredBy    *int64    `json:"referred_by"`
	IsAgent       bool      `gorm:"default:false" json:"is_agent"`
	AgentBalance  float64   `gorm:"default:0" json:"agent_balance"`
	CreatedAt     time.Time `json:"created_at"`
	LastActive    time.Time `json:"last_active"`
	IsBot        bool      `gorm:"default:false;index"`
}

// Game represents a BINGO game round
type Game struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Status           string    `gorm:"default:'waiting'" json:"status"` // waiting, calling, finished, cancelled
	StakeAmount      float64   `json:"stake_amount"`
	MaxCardsPerPlayer int      `gorm:"default:2" json:"max_cards_per_player"`
	MaxPlayers       int       `gorm:"default:400" json:"max_players"`
	CalledNumbers pq.Int64Array `gorm:"type:integer[]"`
	WinnerUserID     *int64    `json:"winner_user_id"`
	WinnerPrize      float64   `gorm:"default:0" json:"winner_prize"`
	TotalPool        float64   `gorm:"default:0" json:"total_pool"`
	StartedAt        *time.Time `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// Card represents a BINGO card purchased for a game
type Card struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	GameID         uuid.UUID `gorm:"type:uuid;index" json:"game_id"`
	UserID         int64     `json:"user_id"`
	CardNumber     int       `json:"card_number"` // Display number 1-400
	CardData       CardJSON  `gorm:"type:jsonb" json:"card_data"`
	IsWinner       bool      `gorm:"default:false" json:"is_winner"`
MarkedNumbers pq.Int64Array `gorm:"type:integer[];default:'{}'" json:"marked_numbers"`
	CreatedAt      time.Time `json:"created_at"`
	Status string `gorm:"default:'reserved'"` // "reserved" | "active" | "winner"
}

// CardJSON represents the BINGO card structure
type CardJSON struct {
	B      []int  `json:"B"`
	I      []int  `json:"I"`
	N      []*int `json:"N"` // null for free space
	G      []int  `json:"G"`
	O      []int  `json:"O"`
	CardID int    `json:"card_id"`
}
// Value converts CardJSON to JSON for PostgreSQL jsonb
func (c CardJSON) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *CardJSON) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, c)

	case string:
		return json.Unmarshal([]byte(v), c)

	default:
		return fmt.Errorf("unsupported CardJSON scan type: %T", value)
	}
}

// Transaction for deposits/withdrawals
type Transaction struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"` // deposit, withdraw, stake, win, referral, agent_commission
	Amount    float64   `json:"amount"`
	Status    string    `gorm:"default:'pending'" json:"status"` // pending, completed, failed
	Method    string    `json:"method"` // telebirr, agent, system
	Reference   string    `gorm:"type:text;uniqueIndex:index:idx_transactions_reference"`
	Metadata  *string   `gorm:"type:jsonb" json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	Description string `json:"description"`
}

// GamePlayer tracks who joined a game
type GamePlayer struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	GameID     uuid.UUID `gorm:"type:uuid;index" json:"game_id"`
	UserID     int64     `json:"user_id"`
	CardsCount int       `gorm:"default:0" json:"cards_count"`
	TotalStake float64   `gorm:"default:0" json:"total_stake"`
	JoinedAt   time.Time `json:"joined_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	IsBot      bool      `gorm:"default:false"`
}

// internal/models/models.go - Add this

// BotSettings stores bot configuration
type RobotBotSettings struct {
	ID            uint   `gorm:"primaryKey"`
	DesiredCount  int    `gorm:"default:20"`
	UpdatedAt     time.Time
}