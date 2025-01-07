// handlers/auth.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"telegram_bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	Bot       *tgbotapi.BotAPI
	Admins    map[int64]bool
	DB        database.DBInterface
	AdminMenu tgbotapi.ReplyKeyboardMarkup
}

// Конструктор для Handler
func NewHandler(bot *tgbotapi.BotAPI, db database.DBInterface, admins map[int64]bool) *Handler {
	return &Handler{
		Bot:    bot,
		Admins: admins,
		DB:     db,
	}
}

func (h *Handler) Start(ctx context.Context, update tgbotapi.Update) {
	user := update.Message.From

	// Проверка, есть ли пользователь в БД
	var existingUserID int
	err := h.DB.QueryRow(ctx, "SELECT id FROM users WHERE telegram_id=$1", user.ID).Scan(&existingUserID)
	if err != nil {
		// Если пользователя нет, добавить его
		_, err = h.DB.Exec(ctx, "INSERT INTO users (telegram_id, username) VALUES ($1, $2)", user.ID, user.UserName)
		if err != nil {
			log.Println("Ошибка при добавлении пользователя:", err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при регистрации. Пожалуйста, попробуйте позже.")
			h.Bot.Send(msg)
			return
		}
	}

	// Создание клавиатуры
	var keyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Показать баланс"),
			tgbotapi.NewKeyboardButton("Личный кабинет"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Взять задание"),
			tgbotapi.NewKeyboardButton("Выполненные задания"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Вывести средства"),
			tgbotapi.NewKeyboardButton("Обратиться в техподдержку"),
		),
	)

	AdminMenu := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить задание"),
			tgbotapi.NewKeyboardButton("Проверить задания"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Главное меню"),
		),
	)

	userID := update.Message.From.ID
	var msg tgbotapi.MessageConfig

	if h.IsAdmin(int64(userID)) {
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Добро пожаловать, администратор!")
		msg.ReplyMarkup = AdminMenu
	} else {
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Добро пожаловать!")
		msg.ReplyMarkup = keyboard
	}

	h.Bot.Send(msg)

	msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Добро пожаловать! Что вы хотите сделать?")
	msg.ReplyMarkup = keyboard
	h.Bot.Send(msg)
}

func (h *Handler) IsAdmin(telegramID int64) bool {
	return h.Admins[telegramID]
}

func (h *Handler) HandleSupport(ctx context.Context, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Свяжитесь с нашей службой поддержки по адресу @support.")
	h.Bot.Send(msg)
}

func (h *Handler) HandleShowAccount(ctx context.Context, update tgbotapi.Update) {
	userID := update.Message.From.ID

	// Получение данных пользователя из базы данных
	user, err := h.DB.GetUserByID(ctx, userID)
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
	msg.ParseMode = "Markdown"

	sentMsg, err := h.Bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка при отправке сообщения: %v", err)
		// Дополнительная обработка ошибки, например, уведомление администратора
		return
	}

	log.Printf("Сообщение отправлено успешно: %v", sentMsg.MessageID)

}

// Другие функции авторизации можно добавить здесь
