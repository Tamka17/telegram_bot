// handlers/balance.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"telegram_bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) ShowBalance(ctx context.Context, chatID int64, telegramID int64) {
	var balance float64
	err := h.DB.QueryRowContext(ctx, "SELECT balance FROM users WHERE telegram_id=$1", telegramID).Scan(&balance)
	if err != nil {
		log.Println("Ошибка при получении баланса:", err)
		msg := tgbotapi.NewMessage(chatID, "Не удалось получить баланс.")
		h.Bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ваш текущий баланс: %.2f руб.", balance))
	h.Bot.Send(msg)
}

func (h *Handler) HandleBalanceCommand(ctx context.Context, update tgbotapi.Update) {
	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	h.ShowBalance(ctx, chatID, telegramID)
}

func (h *Handler) HandleWithdrawRequest(ctx context.Context, update tgbotapi.Update) {
	userID := update.Message.From.ID

	// Получение баланса пользователя
	user, err := h.DB.GetUserByTelegramID(ctx, userID)
	if err != nil || user.Balance <= 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас недостаточно средств для вывода.")
		h.Bot.Send(msg)
		return
	}

	// Запрос номера карты у пользователя
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, введите номер вашей карты для вывода средств.")
	h.Bot.Send(msg)

	// Установка состояния пользователя
	h.DB.SetUserState(ctx, userID, string(models.StateAwaitingCardNumder))
}

func (h *Handler) HandleCardNumberReceived(ctx context.Context, update tgbotapi.Update) {
	userID := update.Message.From.ID
	cardNumber := update.Message.Text

	// Валидация номера карты (при необходимости)
	// ...

	// Получение баланса пользователя
	user, err := h.DB.GetUserByTelegramID(ctx, userID)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.Bot.Send(msg)
		return
	}

	if user.Balance <= 400 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас недостаточно средств для вывода.")
		h.Bot.Send(msg)
		return
	}

	// Отправка информации админу
	adminMessage := fmt.Sprintf(
		"📥 *Запрос на вывод средств*\n\n"+
			"👤 *Пользователь:* %d\n"+
			"💰 *Сумма:* %.2f руб.\n"+
			"💳 *Номер карты:* %s",
		userID,
		user.Balance,
		cardNumber,
	)
	adminMsg := tgbotapi.NewMessage(7113548539, adminMessage)
	adminMsg.ParseMode = "Markdown"
	h.Bot.Send(adminMsg)

	// Обнуление баланса пользователя
	err = h.DB.SetUserBalance(ctx, userID, 0)
	if err != nil {
		// Обработка ошибки
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось обнулить ваш баланс. Пожалуйста, обратитесь в техподдержку.")
		h.Bot.Send(msg)
		return
	}

	// Сброс состояния пользователя
	h.DB.SetUserState(ctx, userID, string(models.StateNone))

	// Уведомление пользователя
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ваш запрос на вывод средств отправлен администратору.")
	h.Bot.Send(msg)
}
