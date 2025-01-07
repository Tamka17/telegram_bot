// handlers/admin.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"telegram_bot/models"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) AddTask(ctx context.Context, update tgbotapi.Update) {
	if !h.IsAdmin(update.Message.From.ID) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет прав для этой команды.")
		h.Bot.Send(msg)
		return
	}

	// Предположим, что администратор отправляет задание в формате "/addtask описание|ссылка"
	msgParts := strings.SplitN(update.Message.Text, " ", 2)
	if len(msgParts) != 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат. Используйте: /addtask описание|ссылка")
		h.Bot.Send(msg)
		return
	}

	taskParts := strings.SplitN(msgParts[1], "|", 2)
	if len(taskParts) != 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат. Используйте: /addtask описание|ссылка")
		h.Bot.Send(msg)
		return
	}

	description := taskParts[0]
	link := taskParts[1]

	_, err := h.DB.Exec(ctx, "INSERT INTO tasks (description, link) VALUES ($1, $2)", description, link)
	if err != nil {
		log.Println("Ошибка при добавлении задания:", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось добавить задание.")
		h.Bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Задание успешно добавлено.")
	h.Bot.Send(msg)
}

func (h *Handler) ViewCompletedTasks(ctx context.Context, update tgbotapi.Update) {
	if !h.IsAdmin(update.Message.From.ID) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет прав для этой команды.")
		h.Bot.Send(msg)
		return
	}

	rows, err := h.DB.Query(ctx, `
        SELECT users.username, tasks.description, user_tasks.status, user_tasks.screenshots, user_tasks.id
        FROM user_tasks
        JOIN users ON user_tasks.user_id = users.id
        JOIN tasks ON user_tasks.task_id = tasks.id
        WHERE user_tasks.status = 'completed'
    `)
	if err != nil {
		log.Println("Ошибка при получении выполненных заданий:", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось получить выполненные задания.")
		h.Bot.Send(msg)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var username, description, status string
		var screenshotsData []byte
		var userTaskID int
		err := rows.Scan(&username, &description, &status, &screenshotsData, &userTaskID)
		if err != nil {
			log.Println("Ошибка при сканировании выполненного задания:", err)
			continue
		}

		var screenshots []string
		err = json.Unmarshal(screenshotsData, &screenshots)
		if err != nil {
			log.Println("Ошибка при разборе скриншотов:", err)
			continue
		}

		response := fmt.Sprintf("Пользователь: %s\nЗадание: %s\nСтатус: %s\n", username, description, status)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		go func() {
			h.Bot.Send(msg)
		}()

		// Отправляем скриншоты
		for _, screenshot := range screenshots {
			photoMsg := tgbotapi.NewPhoto(update.Message.Chat.ID, tgbotapi.FileURL(screenshot))
			h.Bot.Send(photoMsg)
		}

		// Добавляем кнопки для проверки задания
		approveData := fmt.Sprintf("verify_%d_yes", userTaskID)
		rejectData := fmt.Sprintf("verify_%d_no", userTaskID)
		approveButton := tgbotapi.NewInlineKeyboardButtonData("Одобрить", approveData)
		rejectButton := tgbotapi.NewInlineKeyboardButtonData("Отклонить", rejectData)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(approveButton, rejectButton))
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Проверить задание:")
		msg.ReplyMarkup = keyboard
		go func() {
			h.Bot.Send(msg)
		}()
	}
}

func (h *Handler) HandleVerificationAction(ctx context.Context, update tgbotapi.Update) {
	callback := update.CallbackQuery
	data := callback.Data

	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}

	action := parts[0]
	userTaskIDStr := parts[1]
	result := parts[2]
	userTaskID, err := strconv.Atoi(userTaskIDStr)
	if err != nil {
		return
	}

	if action != "verify" {
		return
	}

	// Получаем информацию по заданию
	var userID int
	var taskID int
	var taskDescription string
	err = h.DB.QueryRow(ctx, `
        SELECT user_id, task_id, (SELECT description FROM tasks WHERE id = user_tasks.task_id)
        FROM user_tasks
        WHERE id = $1
    `, userTaskID).Scan(&userID, &taskID, &taskDescription)
	if err != nil {
		log.Println("Ошибка при получении информации о задании:", err)
		return
	}

	var telegramID int64
	err = h.DB.QueryRow(ctx, "SELECT telegram_id FROM users WHERE id=$1", userID).Scan(&telegramID)
	if err != nil {
		log.Println("Ошибка при получении Telegram ID пользователя:", err)
		return
	}

	if result == "yes" {
		// Обновляем статус задания
		_, err = h.DB.Exec(ctx, "UPDATE user_tasks SET status='verified_correct' WHERE id=$1", userTaskID)
		if err != nil {
			log.Println("Ошибка при обновлении статуса задания:", err)
			return
		}

		// Добавляем выплату
		amount := 10.0 // Пример суммы выплаты
		_, err = h.DB.Exec(ctx, "UPDATE users SET balance = balance + $1 WHERE id=$2", amount, userID)
		if err != nil {
			log.Println("Ошибка при обновлении баланса пользователя:", err)
			return
		}

		// Добавляем транзакцию
		_, err = h.DB.Exec(ctx, "INSERT INTO transactions (user_id, amount, description) VALUES ($1, $2, $3)", userID, amount, "Выплата за задание")
		if err != nil {
			log.Println("Ошибка при добавлении транзакции:", err)
			return
		}

		// Уведомляем пользователя
		msg := tgbotapi.NewMessage(telegramID, "Задание выполнено правильно, вы получили выплату.")
		h.Bot.Send(msg)
	} else if result == "no" {
		// Обновляем статус задания
		_, err = h.DB.Exec(ctx, "UPDATE user_tasks SET status='verified_incorrect' WHERE id=$1", userTaskID)
		if err != nil {
			log.Println("Ошибка при обновлении статуса задания:", err)
			return
		}

		// Уведомляем пользователя
		msg := tgbotapi.NewMessage(telegramID, "Задание выполнено неправильно, выплата не производится.")
		h.Bot.Send(msg)
	}

	// Ответ на callback
	callbackConfig := tgbotapi.NewCallback(callback.ID, "Статус задания обновлен.")
	h.Bot.Request(callbackConfig)

	answer := tgbotapi.NewCallback(callback.ID, "Статус задания обновлён.")
	_, err = h.Bot.Request(answer)
	if err != nil {
		log.Println("Ошибка при отправке ответа на CallbackQuery:", err)
	}
}

// Добавьте обработчик команды для администраторов
func (h *Handler) HandleAdminCommands(ctx context.Context, update tgbotapi.Update) {
	if !h.IsAdmin(update.Message.From.ID) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет прав для этой команды.")
		h.Bot.Send(msg)
		return
	}

	text := update.Message.Text
	if strings.HasPrefix(text, "/addtask") {
		h.AddTask(ctx, update)
	} else if text == "/viewcompletedtasks" {
		h.ViewCompletedTasks(ctx, update)
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная администраторская команда.")
		h.Bot.Send(msg)
	}
}

func (h *Handler) HandleAdminAddTask(ctx context.Context, update tgbotapi.Update) {
	// Предложение выбрать тип задания
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Авито"),
			tgbotapi.NewKeyboardButton("Яндекс"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Гугл"),
			tgbotapi.NewKeyboardButton("2GIS"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отменить добавление"),
		),
	)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите тип задания для добавления:")
	msg.ReplyMarkup = keyboard
	h.Bot.Send(msg)

	// Установка состояния администратора
	h.DB.SetUserState(context.Background(), update.Message.From.ID, "awaiting_task_category")
}

func (h *Handler) HandleAdminTaskCategorySelection(ctx context.Context, update tgbotapi.Update) {
	category := update.Message.Text

	if category == "Отменить добавление" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Добавление задания отменено.")
		msg.ReplyMarkup = h.AdminMenu
		h.Bot.Send(msg)
		h.DB.SetUserState(ctx, update.Message.From.ID, "")
		return
	}
	// Сохранение выбранной категории
	h.DB.SetTempData(ctx, update.Message.From.ID, "new_task_category", category)

	// Запрос описания задания
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите описание задания:")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	h.Bot.Send(msg)

	// Обновление состояния администратора
	h.DB.SetUserState(ctx, update.Message.From.ID, "awaiting_task_description")
}

func (h *Handler) HandleAdminTaskDescriptionReceived(ctx context.Context, update tgbotapi.Update) {
	description := update.Message.Text
	adminID := update.Message.From.ID

	// Получение сохраненной категории
	category, err := h.DB.GetTempData(ctx, adminID, "new_task_category")
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось получить категорию задания. Попробуйте снова.")
		if _, err := h.Bot.Send(msg); err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
		return
	}

	// Приведение типа interface{} к string
	categoryStr, ok := category.(string)
	if !ok {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка: Некорректный тип категории.")
		if _, err := h.Bot.Send(msg); err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
		return
	}

	// Создание задания в базе данных
	newTask := models.Task{
		Category:    categoryStr,
		Description: description,
		IsActive:    true,
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		// Добавьте другие необходимые поля...
	}

	// Сохранение задания в базе данных
	err = h.DB.CreateTask(ctx, &newTask)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при создании задания. Попробуйте позже.")
		if _, err := h.Bot.Send(msg); err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
		return
	}

	// Уведомление об успешном добавлении задания
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Задание успешно добавлено!")
	msg.ReplyMarkup = h.AdminMenu
	if _, err := h.Bot.Send(msg); err != nil {
		log.Printf("Ошибка при отправке сообщения: %v", err)
	}

	// Сброс состояния и временных данных
	if err := h.DB.SetUserState(ctx, adminID, ""); err != nil {
		log.Printf("Ошибка при сбросе состояния пользователя: %v", err)
	}
	if err := h.DB.DeleteTempData(ctx, adminID, "new_task_category"); err != nil {
		log.Printf("Ошибка при удалении временных данных: %v", err)
	}
}

func (h *Handler) HandleAdminCheckTasks(ctx context.Context, update tgbotapi.Update) {
	tasks, err := h.DB.GetPendingTasks(ctx) // Задания со статусом "Pending"
	if err != nil || len(tasks) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нет заданий для проверки.")
		h.Bot.Send(msg)
		return
	}

	for _, task := range tasks {
		taskInfo := fmt.Sprintf(
			"👤 *Пользователь:* %d\n"+
				"📄 *Задание:* %s\n"+
				"📝 *Описание:* %s",
			task.ID,
			task.Category,
			task.Description,
		)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, taskInfo)
		msg.ParseMode = "Markdown"

		// Кнопки для одобрения или отклонения задания
		approveButton := tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve_%d", task.ID))
		rejectButton := tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", task.ID))
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(approveButton, rejectButton),
		)
		msg.ReplyMarkup = &keyboard

		h.Bot.Send(msg)
	}
}
