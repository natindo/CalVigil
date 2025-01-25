package services

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jackc/pgx/v5"

	"github.com/natindo/CalVigil/internal/models"
)

// StartNotifier запускает горутину, которая каждые 60 секунд проверяет события.
// Если (start_time - now) <= notify_before и notified=false, отправляем уведомление.
func StartNotifier(bot *tgbotapi.BotAPI, conn *pgx.Conn) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        <-ticker.C
        now := time.Now()

        events, err := findEventsToNotify(conn, now)
        if err != nil {
            log.Println("Ошибка findEventsToNotify:", err)
            continue
        }
        if len(events) == 0 {
            continue
        }

        for _, ev := range events {
            notifyUser(bot, ev)
            if err := markEventNotified(conn, ev.ID); err != nil {
                log.Println("Ошибка markEventNotified:", err)
            }
        }
    }
}

// findEventsToNotify ищет события, для которых пора отправить уведомление.
func findEventsToNotify(conn *pgx.Conn, now time.Time) ([]models.Event, error) {
    rows, err := conn.Query(context.Background(), `
SELECT id, chat_id, title, start_time, end_time, notify_before, notified
FROM events
WHERE notified = false
  AND start_time > $1
  AND (start_time - $1) <= (notify_before * INTERVAL '1 minute')
`, now)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []models.Event
    for rows.Next() {
        var e models.Event
        err := rows.Scan(&e.ID, &e.ChatID, &e.Title, &e.StartTime, &e.EndTime, &e.NotifyBefore, &e.Notified)
        if err != nil {
            return nil, err
        }
        result = append(result, e)
    }
    return result, nil
}

func markEventNotified(conn *pgx.Conn, eventID int) error {
    _, err := conn.Exec(context.Background(), `
UPDATE events
SET notified = true
WHERE id = $1
`, eventID)
    return err
}

func notifyUser(bot *tgbotapi.BotAPI, ev models.Event) {
    mins := ev.NotifyBefore
    startStr := ev.StartTime.Format("15:04")
    endStr := ev.EndTime.Format("15:04")

    text := fmt.Sprintf("Напоминание!\nЧерез %d мин начнётся событие:\n%s\nВремя: %s - %s",
        mins, ev.Title, startStr, endStr)
    msg := tgbotapi.NewMessage(ev.ChatID, text)
    bot.Send(msg)
}