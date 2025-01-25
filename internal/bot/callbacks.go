package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jackc/pgx/v5"

	"github.com/username/planfreebot/internal/models"
	"github.com/username/planfreebot/internal/services"
)

// HandleCallbackQuery обрабатывает клики по inline-кнопкам
func HandleCallbackQuery(bot *tgbotapi.BotAPI, dbConn *pgx.Conn, cq *tgbotapi.CallbackQuery) {
    chatID := cq.Message.Chat.ID
    data := cq.Data

    switch data {
    case "date_today":
        handleDateToday(bot, chatID, cq)
    case "date_tomorrow":
        handleDateTomorrow(bot, chatID, cq)
    case "delete_all_today":
        handleDeleteAllToday(bot, dbConn, chatID, cq)
    default:
        // Если callback_data не узнаём, сообщим пользователю
        bot.AnswerCallbackQuery(tgbotapi.NewCallback(cq.ID, "Неизвестное действие"))
    }
}

func handleDateToday(bot *tgbotapi.BotAPI, chatID int64, cq *tgbotapi.CallbackQuery) {
    state, ok := userCreationState[chatID]
    if !ok || state.Step != 1 {
        bot.AnswerCallbackQuery(tgbotapi.NewCallback(cq.ID, "Нет активного создания или неверный шаг."))
        return
    }

    // Устанавливаем дату = сегодня
    state.SelectedDate = time.Now()
    state.Step = 2
    bot.AnswerCallbackQuery(tgbotapi.NewCallback(cq.ID, "Вы выбрали сегодня."))
    sendNextStep(bot, chatID, state)
}

func handleDateTomorrow(bot *tgbotapi.BotAPI, chatID int64, cq *tgbotapi.CallbackQuery) {
    state, ok := userCreationState[chatID]
    if !ok || state.Step != 1 {
        bot.AnswerCallbackQuery(tgbotapi.NewCallback(cq.ID, "Нет активного создания или неверный шаг."))
        return
    }

    // Устанавливаем дату = завтра
    state.SelectedDate = time.Now().Add(24 * time.Hour)
    state.Step = 2
    bot.AnswerCallbackQuery(tgbotapi.NewCallback(cq.ID, "Вы выбрали завтра."))
    sendNextStep(bot, chatID, state)
}

func handleDeleteAllToday(bot *tgbotapi.BotAPI, dbConn *pgx.Conn, chatID int64, cq *tgbotapi.CallbackQuery) {
    bot.AnswerCallbackQuery(tgbotapi.NewCallback(cq.ID, "")) // Закрыть «часовые песочки» для пользователя

    err := services.DeleteAllToday(dbConn, chatID, time.Now())
    if err != nil {
        bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка при удалении: %v", err)))
        return
    }
    bot.Send(tgbotapi.NewMessage(chatID, "Все сегодняшние события удалены."))
}

// -------------------------------------------------------------------
// Обработчик пошаговых сообщений (handleCreationSteps)
// -------------------------------------------------------------------

func handleCreationSteps(bot *tgbotapi.BotAPI, dbConn *pgx.Conn, msg *tgbotapi.Message) {
    chatID := msg.Chat.ID
    state, ok := userCreationState[chatID]
    if !ok {
        // Нет активного «диалога» — выходим
        return
    }

    switch state.Step {
    case 1:
        // Пользователь должен был ввести дату (YYYY-MM-DD),
        // если не выбрал «Сегодня/Завтра» Inline-кнопками
        dateStr := strings.TrimSpace(msg.Text)
        parsed, err := time.Parse("2006-01-02", dateStr)
        if err != nil {
            bot.Send(tgbotapi.NewMessage(chatID, "Не удалось распознать дату, формат YYYY-MM-DD."))
            return
        }
        state.SelectedDate = parsed
        state.Step = 2
        sendNextStep(bot, chatID, state)

    case 2:
        // Ожидаем время начала в формате HH:MM
        timeStr := strings.TrimSpace(msg.Text)
        t, err := time.Parse("15:04", timeStr)
        if err != nil {
            bot.Send(tgbotapi.NewMessage(chatID, "Некорректный формат времени (HH:MM)."))
            return
        }
        // Собираем дату и время
        dt := time.Date(
            state.SelectedDate.Year(),
            state.SelectedDate.Month(),
            state.SelectedDate.Day(),
            t.Hour(), t.Minute(), 0, 0, time.Local,
        )
        state.SelectedStart = dt
        state.Step = 3
        sendNextStep(bot, chatID, state)

    case 3:
        // Ожидаем ввод длительности в минутах (целое число)
        durStr := strings.TrimSpace(msg.Text)
        durMins, err := strconv.Atoi(durStr)
        if err != nil {
            bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, введите число — длительность в минутах."))
            return
        }
        state.Duration = time.Duration(durMins) * time.Minute
        state.Step = 4
        sendNextStep(bot, chatID, state)

    case 4:
        // Ожидаем ввод "за сколько минут уведомлять"
        notifyStr := strings.TrimSpace(msg.Text)
        notifyMins, err := strconv.Atoi(notifyStr)
        if err != nil {
            bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, введите число — за сколько минут уведомлять."))
            return
        }
        if notifyMins < 0 {
            notifyMins = 0
        }
        state.NotifyBefore = notifyMins
        state.Step = 5
        sendNextStep(bot, chatID, state)

    case 5:
        // Ожидаем название события
        state.Title = strings.TrimSpace(msg.Text)
        // Теперь у нас есть все данные — создаём событие в БД
        endTime := state.SelectedStart.Add(state.Duration)

        ev := models.Event{
            ChatID:       chatID,
            Title:        state.Title,
            StartTime:    state.SelectedStart,
            EndTime:      endTime,
            NotifyBefore: state.NotifyBefore,
            Notified:     false,
        }

        id, err := services.InsertEvent(dbConn, ev)
        if err != nil {
            log.Println("Ошибка InsertEvent:", err)
            bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка при сохранении события."))
            delete(userCreationState, chatID) // Сбрасываем состояние
            return
        }

        // Сообщаем пользователю об успехе
        summary := fmt.Sprintf(
            "Событие создано (ID=%d):\n%s\nНачало: %s\nДлительность: %d минут\nУведомлять за %d мин",
            id,
            state.Title,
            state.SelectedStart.Format("2006-01-02 15:04"),
            int(state.Duration.Minutes()),
            state.NotifyBefore,
        )
        bot.Send(tgbotapi.NewMessage(chatID, summary))

        // Сбрасываем состояние
        delete(userCreationState, chatID)

    default:
        // Неизвестный шаг — на всякий случай сбросим
        delete(userCreationState, chatID)
    }
}

// sendNextStep — отправляет сообщение, что ожидается на следующем шаге
func sendNextStep(bot *tgbotapi.BotAPI, chatID int64, state *models.CreationState) {
    switch state.Step {
    case 2:
        bot.Send(tgbotapi.NewMessage(chatID, "Введите время начала (HH:MM):"))
    case 3:
        bot.Send(tgbotapi.NewMessage(chatID, "Введите длительность события в минутах:"))
    case 4:
        bot.Send(tgbotapi.NewMessage(chatID, "Введите, за сколько минут до начала напоминать:"))
    case 5:
        bot.Send(tgbotapi.NewMessage(chatID, "Введите название события:"))
    }
}