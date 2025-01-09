// handlers/transactions.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) ShowTransactionHistory(ctx context.Context, chatID int64, telegramID int64) {
	var userID int
	err := h.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE telegram_id=$1", telegramID).Scan(&userID)
	if err != nil {
		log.Println("Ошибка при получении user ID:", err)
		msg := tgbotapi.NewMessage(chatID, "Не удалось найти ваш профиль.")
		h.Bot.Send(msg)
		return
	}

	rows, err := h.DB.QueryContext(ctx, `
        SELECT amount, description, created_at
        FROM transactions
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT 10
    `, userID)
	if err != nil {
		log.Println("Ошибка при получении транзакций:", err)
		msg := tgbotapi.NewMessage(chatID, "Не удалось получить историю транзакций.")
		h.Bot.Send(msg)
		return
	}
	defer rows.Close()

	var response string
	for rows.Next() {
		var amount float64
		var description, createdAt string
		err := rows.Scan(&amount, &description, &createdAt)
		if err != nil {
			log.Println("Ошибка при сканировании транзакции:", err)
			continue
		}
		response += fmt.Sprintf("%s: %.2f руб. — %s\n", createdAt, amount, description)
	}

	if response == "" {
		response = "У вас пока нет транзакций."
	}

	msg := tgbotapi.NewMessage(chatID, response)
	h.Bot.Send(msg)
}

func (h *Handler) HandleTransactionHistory(ctx context.Context, update tgbotapi.Update) {
	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	h.ShowTransactionHistory(ctx, chatID, telegramID)
}

func (h *Handler) HandleNextTaskStep(ctx context.Context, update tgbotapi.Update) {
	userID := update.Message.From.ID

	// Проверяем, доступен ли следующий шаг
	availableAt, err := h.DB.GetUserAvailableAt(ctx, userID)
	if err == nil && time.Now().Before(availableAt) {
		remaining := time.Until(availableAt).Round(time.Second)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Пожалуйста, подождите еще %v, прежде чем перейти к следующему шагу.", remaining))
		h.Bot.Send(msg)
		return
	}

	// Далее логика перехода к следующему шагу
	// ...
}

func (h *Handler) StartWaitingPeriod(ctx context.Context, userID int64, delay time.Duration) {
	h.DB.SetUserAvailableAt(ctx, userID)
}
