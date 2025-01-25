package config

import (
	"log"
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

    // Значения по умолчанию (на случай, если env-переменные не заданы)
    if cfg.TelegramToken == "" {
        log.Println("TELEGRAM_BOT_TOKEN не задан в окружении")
        cfg.TelegramToken = "CHANGE_ME"
    }
    if cfg.DatabaseURL == "" {
        log.Println("DATABASE_URL не задан в окружении, используем дефолтную строку подключения")
        cfg.DatabaseURL = "user=myuser password=mypass dbname=mydb host=localhost port=5432 sslmode=disable"
    }

    return cfg
}