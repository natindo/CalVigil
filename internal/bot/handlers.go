package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jackc/pgx/v5"

	"github.com/natindo/CalVigil/internal/models"
	"github.com/natindo/CalVigil/internal/services"
)

// userCreationState хранит в памяти шаги создания события. Для продакшена можно сохранять в БД.
var userCreationState = make(map[int64]*models.CreationState)

func handleCommand(bot *tgbotapi.BotAPI, dbConn *pgx.Conn, msg *tgbotapi.Message) {
    switch msg.Command() {
    case "start":
        cmdStart(bot, msg)
    case "help":
        cmdHelp(bot, msg)
    case "list":
        cmdList(bot, dbConn, msg)
    case "create":
        cmdCreate(bot, msg)
    case "delete":
        cmdDelete(bot, dbConn, msg)
    case "update":
        cmdUpdate(bot, dbConn, msg)
    default:
        unknownCommand(bot, msg)
    }
}

func cmdStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
    text := "Привет! Я бот-планировщик.\n" +
        "Доступные команды:\n" +
        "/create — пошагово создать событие\n" +
        "/list — показать события на сегодня\n" +
        "/delete <id> — удалить событие\n" +
        "/update <id> — изменить событие\n" +
        "/help — справка"
    bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

func cmdHelp(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
    text := "Справка:\n" +
        "/create — начать диалог по созданию события\n" +
        "/list — показать события на сегодня\n" +
        "/delete <id> — удалить событие\n" +
        "/update <id> — изменить событие\n"
    bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

func cmdList(bot *tgbotapi.BotAPI, dbConn *pgx.Conn, msg *tgbotapi.Message) {
    evs, err := services.GetEventsForToday(dbConn, msg.Chat.ID, time.Now())
    if err != nil {
        log.Println("Ошибка при GetEventsForToday:", err)
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ошибка при получении списка событий"))
        return
    }
    if len(evs) == 0 {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "На сегодня нет событий."))
        return
    }

    var sb strings.Builder
    sb.WriteString("Ваши события на сегодня:\n")
    for i, e := range evs {
        startStr := e.StartTime.Format("15:04")
        endStr := e.EndTime.Format("15:04")
        sb.WriteString(fmt.Sprintf("%d) ID=%d | %s (%s - %s)\n", i+1, e.ID, e.Title, startStr, endStr))
    }

    // Пример inline-кнопки: «Удалить все события за сегодня»
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Удалить все за сегодня", "delete_all_today"),
        ),
    )
    message := tgbotapi.NewMessage(msg.Chat.ID, sb.String())
    message.ReplyMarkup = keyboard
    bot.Send(message)
}

func cmdCreate(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
    // Инициализируем состояние
    userCreationState[msg.Chat.ID] = &models.CreationState{
        Step:         1,
        NotifyBefore: 5, // по умолчанию 5 минут
    }

    text := "Приступим к созданию события.\nВыберите дату или введите её (формат YYYY-MM-DD):"
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Сегодня", "date_today"),
            tgbotapi.NewInlineKeyboardButtonData("Завтра", "date_tomorrow"),
        ),
    )
    msgOut := tgbotapi.NewMessage(msg.Chat.ID, text)
    msgOut.ReplyMarkup = keyboard
    bot.Send(msgOut)
}

func cmdDelete(bot *tgbotapi.BotAPI, dbConn *pgx.Conn, msg *tgbotapi.Message) {
    args := msg.CommandArguments()
    if args == "" {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Укажите ID события: /delete 123"))
        return
    }
    id, err := strconv.Atoi(args)
    if err != nil {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Некорректный ID."))
        return
    }

    err = services.DeleteEvent(dbConn, msg.Chat.ID, id)
    if err != nil {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при удалении: %v", err)))
        return
    }
    bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Событие удалено."))
}

func cmdUpdate(bot *tgbotapi.BotAPI, dbConn *pgx.Conn, msg *tgbotapi.Message) {
    args := msg.CommandArguments()
    if args == "" {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Укажите ID события: /update 123"))
        return
    }
    id, err := strconv.Atoi(args)
    if err != nil {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Некорректный ID."))
        return
    }

    ev, err := services.GetEventByID(dbConn, msg.Chat.ID, id)
    if err != nil {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Ошибка при получении события: %v", err)))
        return
    }
    if ev == nil {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Событие не найдено."))
        return
    }

    // «Удаляем» старое событие, чтобы пересоздать (упрощённо)
    err = services.DeleteEvent(dbConn, msg.Chat.ID, id)
    if err != nil {
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ошибка при подготовке к редактированию."))
        return
    }

    // Инициализируем state
    userCreationState[msg.Chat.ID] = &models.CreationState{
        Step:          1,
        SelectedDate:  ev.StartTime,
        SelectedStart: ev.StartTime,
        Duration:      ev.EndTime.Sub(ev.StartTime),
        NotifyBefore:  ev.NotifyBefore,
        Title:         ev.Title,
    }

    text := "Обновление события.\nСначала выберите/введите дату (YYYY-MM-DD)."
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Сегодня", "date_today"),
            tgbotapi.NewInlineKeyboardButtonData("Завтра", "date_tomorrow"),
        ),
    )
    msgOut := tgbotapi.NewMessage(msg.Chat.ID, text)
    msgOut.ReplyMarkup = keyboard
    bot.Send(msgOut)
}

func unknownCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
    bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Неизвестная команда. Используйте /help"))
}