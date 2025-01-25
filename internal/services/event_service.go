package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/natindo/CalVigil/internal/models"
)

// InsertEvent вставляет новое событие в БД и возвращает его ID
func InsertEvent(conn *pgx.Conn, ev models.Event) (int, error) {
    var newID int
    err := conn.QueryRow(context.Background(), `
INSERT INTO events (chat_id, title, start_time, end_time, notify_before, notified)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`, ev.ChatID, ev.Title, ev.StartTime, ev.EndTime, ev.NotifyBefore, ev.Notified).Scan(&newID)
    if err != nil {
        return 0, err
    }
    return newID, nil
}

// DeleteEvent удаляет событие по ID (только если chat_id совпадает)
func DeleteEvent(conn *pgx.Conn, chatID int64, eventID int) error {
    _, err := conn.Exec(context.Background(), `
DELETE FROM events
WHERE chat_id = $1 AND id = $2
`, chatID, eventID)
    return err
}

// GetEventByID возвращает событие, если оно принадлежит chatID
func GetEventByID(conn *pgx.Conn, chatID int64, eventID int) (*models.Event, error) {
    row := conn.QueryRow(context.Background(), `
SELECT id, chat_id, title, start_time, end_time, notify_before, notified
FROM events
WHERE chat_id = $1 AND id = $2
`, chatID, eventID)

    var e models.Event
    err := row.Scan(
        &e.ID,
        &e.ChatID,
        &e.Title,
        &e.StartTime,
        &e.EndTime,
        &e.NotifyBefore,
        &e.Notified,
    )
    if err != nil {
        if err.Error() == "no rows in result set" {
            return nil, nil
        }
        return nil, err
    }
    return &e, nil
}

// GetEventsForToday возвращает события, которые начинаются в течение текущих суток
func GetEventsForToday(conn *pgx.Conn, chatID int64, now time.Time) ([]models.Event, error) {
    startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    endOfDay := startOfDay.Add(24 * time.Hour)

    rows, err := conn.Query(context.Background(), `
SELECT id, chat_id, title, start_time, end_time, notify_before, notified
FROM events
WHERE chat_id = $1
  AND start_time >= $2
  AND start_time <  $3
ORDER BY start_time
`, chatID, startOfDay, endOfDay)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []models.Event
    for rows.Next() {
        var e models.Event
        if err := rows.Scan(
            &e.ID, &e.ChatID, &e.Title,
            &e.StartTime, &e.EndTime,
            &e.NotifyBefore, &e.Notified,
        ); err != nil {
            return nil, err
        }
        result = append(result, e)
    }
    return result, nil
}

// DeleteAllToday пример удаления всех сегодняшних событий
func DeleteAllToday(conn *pgx.Conn, chatID int64, now time.Time) error {
    startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    endOfDay := startOfDay.Add(24 * time.Hour)

    _, err := conn.Exec(context.Background(), `
DELETE FROM events
WHERE chat_id = $1
  AND start_time >= $2
  AND start_time <  $3
`, chatID, startOfDay, endOfDay)
    return err
}