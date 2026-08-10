package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL  string
	RedisURL     string
	BotToken     string
	AgentBotToken  string // ✅ New field
	WebAppURL    string
	JWTSecret    string
	Bot          BotConfig `json:"bot"`
	VerifyAPIKey string    `env:"VERIFY_API_KEY" default:""`
	BabiBingoPhone string `env:"BABIBINGO_PHONE" default:"0940072277"` // ✅ Add this too
	AdminBotToken   string `env:"ADMIN_BOT_TOKEN" default:""`
    AdminIDs        string `env:"ADMIN_IDS" default:"1929724270"`
    AutoApprove     bool   `env:"AUTO_APPROVE" default:"false"`
    NotifyOnApply   bool   `env:"NOTIFY_ON_APPLY" default:"true"`
    NotifyOnDeposit bool   `env:"NOTIFY_ON_DEPOSIT" default:"true"`
    NotifyOnWithdraw bool  `env:"NOTIFY_ON_WITHDRAW" default:"true"`
}

type BotConfig struct {
	Enabled         bool `json:"enabled"`
	MinBotsPerGame  int  `json:"min_bots_per_game"`
	MaxBotsPerGame  int  `json:"max_bots_per_game"`
	BotsPerTick     int  `json:"bots_per_tick"`
	ReserveInterval int  `json:"reserve_interval"` // seconds
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/babibingo?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		BotToken:       getEnv("TELEGRAM_BOT_TOKEN", ""),
		AgentBotToken:  getEnv("AGENT_BOT_TOKEN", ""),
		WebAppURL:      getEnv("WEBAPP_URL", "https://your-domain.com"),
		JWTSecret:      getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		VerifyAPIKey:   getEnv("VERIFY_API_KEY", ""), // ✅ Add this
		BabiBingoPhone: getEnv("BABIBINGO_PHONE", "0940072277"), // ✅ Add this
		AdminBotToken:   getEnv("ADMIN_BOT_TOKEN", ""),
        AdminIDs:        getEnv("ADMIN_IDS", ""),
        AutoApprove:     getEnv("AUTO_APPROVE", "false") == "true",
        NotifyOnApply:   getEnv("NOTIFY_ON_APPLY", "true") == "true",
        NotifyOnDeposit: getEnv("NOTIFY_ON_DEPOSIT", "true") == "true",
        NotifyOnWithdraw: getEnv("NOTIFY_ON_WITHDRAW", "true") == "true",
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
func (c *Config) GetAdminIDs() []int64 {
    if c.AdminIDs == "" {
        return []int64{}
    }
    
    var ids []int64
    for _, part := range strings.Split(c.AdminIDs, ",") {
        if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
            ids = append(ids, id)
        }
    }
    return ids
}