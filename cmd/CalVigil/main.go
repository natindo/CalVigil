package main

import (
    "log"
    "os"

    "github.com/username/planfreebot/internal/bot"
    "github.com/username/planfreebot/internal/config"
    "github.com/username/planfreebot/internal/database"
    "github.com/username/planfreebot/internal/services"
)

func main() {
    // 1. Читаем конфиг (из env или из файла — как удобнее)
    cfg := config.LoadConfig()

    // 2. Подключаемся к БД
    dbConn, err := database.ConnectPostgres(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Не удалось подключиться к PostgreSQL: %v", err)
    }
    defer dbConn.Close()

    // 3. Создаём инстанс бота
    botAPI, err := bot.NewBot(cfg.TelegramToken)
    if err != nil {
        log.Fatalf("Ошибка при создании бота: %v", err)
    }

    // 4. Запускаем воркер уведомлений (notifier)
    go services.StartNotifier(botAPI, dbConn)

    // 5. Запускаем основной цикл обработки
    if err := bot.Run(botAPI, dbConn); err != nil {
        log.Fatalf("Ошибка запуска бота: %v", err)
    }
}