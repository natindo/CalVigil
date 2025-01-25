package config

import (
	"os"
)

// Config хранит основные настройки приложения.
type Config struct {
	TelegramToken string
	DatabaseURL   string
}

func LoadConfig() *Config {
	cfg := &Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
	}
	return cfg
}
