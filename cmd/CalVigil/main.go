package main

import (
	"context"
	"log"

	"github.com/natindo/CalVigil/internal/bot"
	"github.com/natindo/CalVigil/internal/config"
	"github.com/natindo/CalVigil/internal/database"
	"github.com/natindo/CalVigil/internal/services"
)

func main() {
	// 1. Читаем конфиг (из env или из файла — как удобнее)
	cfg := config.LoadConfig()

	// 2. Подключаемся к БД
	dbConn, err := database.ConnectPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к PostgreSQL: %v", err)
	}
	ctx := context.Background()
	defer dbConn.Close(ctx)

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
