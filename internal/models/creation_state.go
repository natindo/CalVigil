package models

import "time"

// CreationState описывает пошаговое создание/редактирование события.
// Может храниться в памяти, а при желании - в отдельной таблице в БД.
type CreationState struct {
    Step          int
    SelectedDate  time.Time
    SelectedStart time.Time
    Duration      time.Duration
    NotifyBefore  int
    Title         string
}