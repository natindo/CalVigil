package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jackc/pgx/v5"
)

// NewBot инициализирует и возвращает *tgbotapi.BotAPI
func NewBot(token string) (*tgbotapi.BotAPI, error) {
    bot, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }
    // Можно включить Debug-логирование:
    bot.Debug = false
    fmt.Printf("Бот %s успешно инициализирован\n", bot.Self.UserName)
    return bot, nil
}

// Run запускает основной цикл: чтение апдейтов и их обработку
func Run(bot *tgbotapi.BotAPI, dbConn *pgx.Conn) error {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates, err := bot.GetUpdatesChan(u)
    if err != nil {
        return err
    }

    for update := range updates {
        // Inline-кнопки (CallbackQuery)
        if update.CallbackQuery != nil {
            HandleCallbackQuery(bot, dbConn, update.CallbackQuery)
            continue
        }

        // Обычные сообщения
        if update.Message == nil {
            continue
        }

        if update.Message.IsCommand() {
            handleCommand(bot, dbConn, update.Message)
        } else {
            // Возможно, пользователь в процессе пошагового создания
            handleCreationSteps(bot, dbConn, update.Message)
        }
    }
    return nil
}