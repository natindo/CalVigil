package models

import "time"

// Event хранит данные о событии в календаре
type Event struct {
    ID           int
    ChatID       int64
    Title        string
    StartTime    time.Time
    EndTime      time.Time
    NotifyBefore int
    Notified     bool
}