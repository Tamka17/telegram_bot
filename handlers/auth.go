// handlers/auth.go
package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"telegram_bot/database"
	"telegram_bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	Bot           *tgbotapi.BotAPI
	DB            database.DBInterface
	AdminMenu     tgbotapi.ReplyKeyboardMarkup
	Keyboard      tgbotapi.ReplyKeyboardMarkup
	AdminMenuTask tgbotapi.ReplyKeyboardMarkup
	KeyboardTask  tgbotapi.ReplyKeyboardMarkup
}

// Конструктор для Handler
func NewHandler(bot *tgbotapi.BotAPI, db database.DBInterface, admins map[int64]bool) *Handler {
	// Инициализация админского меню задач
	adminMenuTask := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(string(models.CategoryAvito)),
			tgbotapi.NewKeyboardButton(string(models.CategoryYandex)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(string(models.CategoryGoogle)),
			tgbotapi.NewKeyboardButton(string(models.Category2GIS)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отменить добавление"),
		),
	)
	return &Handler{
		Bot:           bot,
		DB:            db,
		AdminMenuTask: adminMenuTask,
	}
}

func (h *Handler) Start(ctx context.Context, update tgbotapi.Update) {
	telegramUser := update.Message.From
	chatID := update.Message.Chat.ID

	// Попытка получить пользователя из базы данных
	var user models.User
	err := h.DB.QueryRowContext(ctx, "SELECT id, telegram_id, admin FROM users WHERE telegram_id=$1", telegramUser.ID).
		Scan(&user.ID, &user.TelegramID, &user.Admin)
	if err != nil {
		if err == sql.ErrNoRows {
			// Пользователь не найден, добавляем его в базу данных
			_, err = h.DB.ExecContext(ctx, "INSERT INTO users (telegram_id, username, admin) VALUES ($1, $2, $3)", telegramUser.ID, telegramUser.UserName, false)
			if err != nil {
				log.Println("Ошибка при добавлении пользователя:", err)
				msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при регистрации. Пожалуйста, попробуйте позже.")
				h.Bot.Send(msg)
				return
			}
			// Устанавливаем статус администратора в false
			user.Admin = false
		} else {
			// Обработка других ошибок
			log.Println("Ошибка при получении пользователя из базы данных:", err)
			msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте позже.")
			h.Bot.Send(msg)
			return
		}
	}

	// Создание клавиатуры
	var UserMenu = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Показать баланс"),
			tgbotapi.NewKeyboardButton("Личный кабинет"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Вывести средства"),
			tgbotapi.NewKeyboardButton("Обратиться в техподдержку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Взять задание"),
		),
	)

	AdminMenu := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить задание"),
			tgbotapi.NewKeyboardButton("Проверить задания"),
		),
	)
	var msg tgbotapi.MessageConfig

	if user.Admin {
		msg = tgbotapi.NewMessage(chatID, "Добро пожаловать, администратор!")
		msg.ReplyMarkup = AdminMenu
	} else {
		msg = tgbotapi.NewMessage(chatID, "Добро пожаловать! Что вы хотите сделать?")
		msg.ReplyMarkup = UserMenu
	}

	h.Bot.Send(msg)
}

func (h *Handler) HandleSupport(ctx context.Context, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Свяжитесь с нашей службой поддержки по адресу @support.")
	h.Bot.Send(msg)
}

func (h *Handler) HandleShowAccount(ctx context.Context, update tgbotapi.Update) {
	userID := update.Message.From.ID

	// Получение данных пользователя из базы данных
	user, err := h.DB.GetUserByTelegramID(ctx, userID)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при получении ваших данных. Пожалуйста, попробуйте позже.")
		h.Bot.Send(msg)
		return
	}

	// Формирование реферальной ссылки
	referralLink := fmt.Sprintf("https://t.me/Co_Work_online_bot?start=%d", userID)

	// Получение статистики рефералов
	referralCount, err := h.DB.GetUserReferralCount(ctx, userID)
	if err != nil {
		referralCount = 0
	}

	// Получение количества выполненных заданий
	completedTasks, err := h.DB.GetCompletedTasksCount(ctx, userID)
	if err != nil {
		completedTasks = 0
	}

	// Формирование сообщения
	accountInfo := fmt.Sprintf(
		"📋 *Личный кабинет*\n\n"+
			"🆔 *Ваш ID:* %d\n"+
			"💰 *Заработано денег:* %.2f руб.\n"+
			"✅ *Выполнено заданий:* %d\n"+
			"🔗 *Ваша реферальная ссылка:*\n%s\n"+
			"👥 *Приглашено рефералов:* %d",
		userID,
		user.Balance,
		completedTasks,
		referralLink,
		referralCount,
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, accountInfo)

	h.Bot.Send(msg)
}

// Другие функции авторизации можно добавить здесь
