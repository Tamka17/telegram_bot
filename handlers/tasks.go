// handlers/tasks.go
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

func CalculateReward(category models.Category) float64 {
	var rewardMap = map[models.Category]float64{
		models.CategoryAvito:  130.0,
		models.CategoryYandex: 25.0,
		models.CategoryGoogle: 25.0,
		models.Category2GIS:   25.0,
	}

	if reward, exists := rewardMap[category]; exists {
		return reward
	}
	return 0.0
}

////////////

func (h *Handler) HandleAssignTask(ctx context.Context, update tgbotapi.Update) {
	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	// Получение user_id
	var userID int
	err := h.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE telegram_id=$1", telegramID).Scan(&userID)
	if err != nil {
		log.Println("Ошибка при получении user ID:", err)
		msg := tgbotapi.NewMessage(chatID, "Не удалось найти ваш профиль.")
		h.Bot.Send(msg)
		return
	}

	// Проверка наличия незавершенного задания
	var existingTaskID int
	err = h.DB.QueryRowContext(ctx, "SELECT task_id FROM user_tasks WHERE user_id=$1 AND status != 'verified_correct' AND status != 'verified_incorrect'", userID).Scan(&existingTaskID)
	if err == nil {
		msg := tgbotapi.NewMessage(chatID, "У вас уже есть незавершенное задание.")
		h.Bot.Send(msg)
		return
	}

	// Получение первого доступного задания
	var taskID int
	err = h.DB.QueryRowContext(ctx, `
        SELECT tasks.id FROM tasks
        LEFT JOIN user_tasks ON tasks.id = user_tasks.task_id
        WHERE tasks.is_active = TRUE
        GROUP BY tasks.id
        HAVING COUNT(user_tasks.id) = 0
        ORDER BY tasks.id
        LIMIT 1
    `).Scan(&taskID)

	if err != nil {
		log.Println("Ошибка при получении задания:", err)
		msg := tgbotapi.NewMessage(chatID, "Задания временно недоступны.")
		h.Bot.Send(msg)
		return
	}

	// Получение деталей задания
	var description, link string
	err = h.DB.QueryRowContext(ctx, "SELECT description, link FROM tasks WHERE id=$1", taskID).Scan(&description, &link)
	if err != nil {
		log.Println("Ошибка при получении деталей задания:", err)
		return
	}

	KeyboardTask := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Авито"),
			tgbotapi.NewKeyboardButton("Яндекс"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Гугл"),
			tgbotapi.NewKeyboardButton("2GIS"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите тип задания для выполнения:")
	msg.ReplyMarkup = KeyboardTask
	h.Bot.Send(msg)

	// Установка состояния администратора
	h.DB.SetUserState(context.Background(), update.Message.From.ID, string(models.StateawaitingTaskCategoryUser))

	h.Bot.Send(msg)
}

func (h *Handler) HandleTaskAction(ctx context.Context, update tgbotapi.Update) {
	callback := update.CallbackQuery
	data := callback.Data

	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return
	}

	action := parts[0]
	taskIDStr := parts[1]
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return
	}

	telegramID := callback.From.ID

	// Получение user_id
	var userID int
	err = h.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE telegram_id=$1", telegramID).Scan(&userID)
	if err != nil {
		log.Println("Ошибка при получении user ID:", err)
		return
	}

	switch action {
	case "starttask":
		// Отправка первого этапа
		h.SendTaskStage(ctx, callback.Message.Chat.ID, userID, taskID)
	case "nextstage":
		// Переход к следующему этапу
		_, err = h.DB.ExecContext(ctx, "UPDATE user_tasks SET current_stage = current_stage + 1, last_updated = NOW() WHERE task_id=$1 AND user_id=$2", taskID, userID)
		if err != nil {
			log.Println("Ошибка при обновлении этапа задания:", err)
			return
		}

		// Запуск таймеров при необходимости
		var currentStage int
		err = h.DB.QueryRowContext(ctx, "SELECT current_stage FROM user_tasks WHERE task_id=$1 AND user_id=$2", taskID, userID).Scan(&currentStage)
		if err != nil {
			log.Println("Ошибка при получении текущего этапа:", err)
			return
		}

		if currentStage == 2 {
			// Тайм-аут в 1 час
			go func() {
				time.Sleep(1 * time.Hour)
				h.NotifyUserStage(ctx, userID, taskID, 2)
			}()
		} else if currentStage == 3 {
			// Тайм-аут в 5 часов
			go func() {
				time.Sleep(5 * time.Hour)
				h.NotifyUserStage(ctx, userID, taskID, 3)
			}()
		}

		// Отправка следующего этапа
		h.SendTaskStage(ctx, callback.Message.Chat.ID, userID, taskID)
	}

	// Ответ на callback
	callbackConfig := tgbotapi.NewCallback(callback.ID, string(models.StateNone))
	h.Bot.Request(callbackConfig)
}

func (h *Handler) SendTaskStage(ctx context.Context, chatID int64, userID int, taskID int) {
	var currentStage int
	err := h.DB.QueryRowContext(ctx, "SELECT current_stage FROM user_tasks WHERE task_id=$1 AND user_id=$2", taskID, userID).Scan(&currentStage)
	if err != nil {
		log.Println("Ошибка при получении текущего этапа:", err)
		return
	}

	var message string
	switch currentStage {
	case 1:
		message = "Первый этап задания. Нажмите 'Далее' после выполнения."
	case 2:
		message = "Выполнили второй пункт? Пришлите скриншот экрана с добавлением объявления в избранное."
	case 3:
		message = "Выполнили третий пункт? Пришлите скриншот с отзывом."
	default:
		message = "Все этапы задания выполнены."
		// Обновить статус задания на completed
		_, err = h.DB.ExecContext(ctx, "UPDATE user_tasks SET status='completed', last_updated=NOW() WHERE task_id=$1 AND user_id=$2", taskID, userID)
		if err != nil {
			log.Println("Ошибка при обновлении статуса задания:", err)
		}
		// Уведомить пользователя о проверке
		h.NotifyUserForVerification(ctx, userID, taskID)
		return
	}

	msg := tgbotapi.NewMessage(chatID, message)
	callbackData := fmt.Sprintf("nextstage_%d", taskID)
	button := tgbotapi.NewInlineKeyboardButtonData("Далее", callbackData)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(button))
	msg.ReplyMarkup = keyboard
	h.Bot.Send(msg)
}

func (h *Handler) NotifyUserStage(ctx context.Context, userID int, taskID int, stage int) {
	var telegramID int64
	err := h.DB.QueryRowContext(ctx, "SELECT telegram_id FROM users WHERE id=$1", userID).Scan(&telegramID)
	if err != nil {
		log.Println("Ошибка при получении Telegram ID:", err)
		return
	}

	var message string
	switch stage {
	case 2:
		message = "Вы можете перейти ко второму этапу задания."
	case 3:
		message = "Вы можете перейти к третьему этапу задания."
	}

	if message != "" {
		msg := tgbotapi.NewMessage(telegramID, message)
		h.Bot.Send(msg)
	}
}

func (h *Handler) NotifyUserForVerification(ctx context.Context, userID int, taskID int) {
	var telegramID int64
	err := h.DB.QueryRowContext(ctx, "SELECT telegram_id FROM users WHERE id=$1", userID).Scan(&telegramID)
	if err != nil {
		log.Println("Ошибка при получении Telegram ID:", err)
		return
	}

	msg := tgbotapi.NewMessage(telegramID, "Ваше задание будет проверено в течение двух дней.")
	h.Bot.Send(msg)
}

///////////////////////////////////////////////////

func (h *Handler) ShowCompletedTasks(ctx context.Context, chatID int64, telegramID int64) {
	var userID int
	err := h.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE telegram_id=$1", telegramID).Scan(&userID)
	if err != nil {
		log.Println("Ошибка при получении user ID:", err)
		msg := tgbotapi.NewMessage(chatID, "Не удалось найти ваш профиль.")
		h.Bot.Send(msg)
		return
	}

	rows, err := h.DB.QueryContext(ctx, `
        SELECT tasks.description, user_tasks.status, user_tasks.last_updated
        FROM user_tasks
        JOIN tasks ON user_tasks.task_id = tasks.id
        WHERE user_tasks.user_id = $1 AND user_tasks.status = ANY($2)ORDER BY user_tasks.last_updated DESC
        LIMIT 10
    `, userID, []string{"verified_correct", "verified_incorrect", "completed"})
	if err != nil {
		log.Println("Ошибка при получении выполненных заданий:", err)
		msg := tgbotapi.NewMessage(chatID, "Не удалось получить выполненные задания.")
		h.Bot.Send(msg)
		return
	}
	defer rows.Close()

	var response string
	for rows.Next() {
		var description, status, lastUpdated string
		err := rows.Scan(&description, &status, &lastUpdated)
		if err != nil {
			log.Println("Ошибка при сканировании задания:", err)
			continue
		}
		response += fmt.Sprintf("Задание: %s\nСтатус: %s\nДата: %s\n\n", description, status, lastUpdated)
	}

	if response == "" {
		response = "У вас нет выполненных заданий."
	}

	msg := tgbotapi.NewMessage(chatID, response)
	h.Bot.Send(msg)
}

func (h *Handler) HandleShowCompletedTasks(ctx context.Context, update tgbotapi.Update) {
	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	h.ShowCompletedTasks(ctx, chatID, telegramID)
}

func (h *Handler) HandleSelectTaskType(ctx context.Context, update tgbotapi.Update) {
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
			tgbotapi.NewKeyboardButton("Назад"),
		),
	)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите тип задания:")
	msg.ReplyMarkup = keyboard
	h.Bot.Send(msg)

	// Установка состояния пользователя
	h.DB.SetUserState(ctx, update.Message.From.ID, "awaiting_task_type")
}

func (h *Handler) HandleCallbackQuery(ctx context.Context, update tgbotapi.Update) {
	callback := update.CallbackQuery
	data := callback.Data

	// Парсинг данных callback
	parts := strings.SplitN(data, "_", 2)
	if len(parts) != 2 {
		h.sendCallbackResponse(callback.ID, "Некорректные данные.")
		return
	}

	action := parts[0]
	taskIDStr := parts[1]
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		h.sendCallbackResponse(callback.ID, "Некорректный ID задания.")
		return
	}

	switch action {
	case "approve":
		// Получение задачи из базы данных для получения категории и вознаграждения
		task, err := h.DB.GetTaskByID(ctx, int64(taskID))
		if err != nil {
			h.sendCallbackResponse(callback.ID, "Ошибка при получении задания.")
			return
		}

		reward := CalculateReward(task.Category)

		// Обновление баланса пользователя
		err = h.DB.UpdateUserBalance(ctx, int64(task.UserID), reward)
		if err != nil {
			h.sendCallbackResponse(callback.ID, "Ошибка при обновлении баланса пользователя.")
			return
		}

		// Обновление статуса задачи
		err = h.DB.UpdateTaskStatus(ctx, int64(taskID), models.StatusApproved)
		if err != nil {
			h.sendCallbackResponse(callback.ID, "Ошибка при обновлении статуса задачи.")
			return
		}

		h.sendCallbackResponse(callback.ID, "Задание одобрено.")

		// Уведомление пользователя
		msg := tgbotapi.NewMessage(int64(task.ID), "Ваше задание одобрено! Вам начислено "+fmt.Sprintf("%.2f", reward)+" руб.")
		h.Bot.Send(msg)

		// Удаление клавиатуры после одобрения
		err = h.removeInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID)
		if err != nil {
			fmt.Printf("Ошибка при удалении inline клавиатуры: %v\n", err)
		}

	case "reject":
		// Обновление статуса задачи
		err = h.DB.UpdateTaskStatus(ctx, int64(taskID), models.StatusRejected)
		if err != nil {
			h.sendCallbackResponse(callback.ID, "Ошибка при отклонении задания.")
			return
		}

		h.sendCallbackResponse(callback.ID, "Задание отклонено.")

		// Удаление клавиатуры после отклонения
		err = h.removeInlineKeyboard(callback.Message.Chat.ID, callback.Message.MessageID)
		if err != nil {
			fmt.Printf("Ошибка при удалении inline клавиатуры: %v\n", err)
		}

	default:
		h.sendCallbackResponse(callback.ID, "Неизвестное действие.")
	}
}

func (h *Handler) sendCallbackResponse(callbackID, message string) {
	callbackConfig := tgbotapi.NewCallback(callbackID, message)
	if _, err := h.Bot.Request(callbackConfig); err != nil {
		fmt.Printf("Ошибка при отправке ответа на CallbackQuery: %v\n", err)
	}
}

// removeInlineKeyboard удаляет inline клавиатуру из сообщения
func (h *Handler) removeInlineKeyboard(chatID int64, messageID int) error {
	editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{})
	_, err := h.Bot.Request(editMsg)
	return err
}

func (h *Handler) StartTaskStep(ctx context.Context, update tgbotapi.Update, delay time.Duration) {

	// Сохранение времени доступности следующего шага
	h.DB.SetUserAvailableAt(ctx, update.Message.From.ID)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Следующий шаг будет доступен через %v минут.", delay.Minutes()))
	h.Bot.Send(msg)
}

func (h *Handler) HandleScreenshot(ctx context.Context, update tgbotapi.Update) {
	photo := update.Message.Photo
	if len(photo) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, отправьте скриншот.")
		h.Bot.Send(msg)
		return
	}

	// Получение FileID самого большого размера фотографии
	fileID := photo[len(photo)-1].FileID

	// Получение объекта File от Telegram
	file, err := h.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		log.Println("Ошибка при получении файла:", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось получить файл скриншота. Попробуйте снова.")
		h.Bot.Send(msg)
		return
	}

	// Получение ссылки на файл
	fileURL := file.Link(h.Bot.Token)

	// Сохранение fileID в базе данных, связав его с текущим заданием пользователя
	err = h.DB.SaveUserTaskScreenshot(ctx, update.Message.From.ID, fileID)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось сохранить скриншот. Попробуйте снова.")
		h.Bot.Send(msg)
		return
	}

	// Сброс состояния пользователя
	err = h.DB.SetUserState(ctx, update.Message.From.ID, "")
	if err != nil {
		log.Println("Ошибка при сбросе состояния пользователя:", err)
		// Можно отправить сообщение пользователю или продолжить
	}

	// Переход к следующему шагу
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Скриншот получен! Можете переходить к следующему шагу.")
	h.Bot.Send(msg)

	// Получение user_id и task_id
	var userID, taskID, currentStage int
	err = h.DB.QueryRowContext(ctx, ` 
        SELECT user_tasks.user_id, user_tasks.task_id, user_tasks.current_stage
        FROM user_tasks
        JOIN users ON users.id = user_tasks.user_id
        WHERE users.telegram_id = $1 AND user_tasks.status = 'in_progress'
        ORDER BY user_tasks.last_updated DESC
        LIMIT 1
   `, update.Message.From.ID).Scan(&userID, &taskID, &currentStage)
	if err != nil {
		log.Println("Ошибка при получении user_id и task_id:", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось найти активное задание.")
		h.Bot.Send(msg)
		return
	}

	// Обновление записи в базе данных с добавлением ссылки на скриншот
	var screenshots []string
	var screenshotsData []byte
	err = h.DB.QueryRowContext(ctx, "SELECT screenshots FROM user_tasks WHERE user_id=$1 AND task_id=$2", userID, taskID).Scan(&screenshotsData)
	if err != nil && err.Error() != "no rows in result set" {
		log.Println("Ошибка при получении скриншотов:", err)
		return
	}

	if len(screenshotsData) > 0 {
		err = json.Unmarshal(screenshotsData, &screenshots)
		if err != nil {
			log.Println("Ошибка при разборе скриншотов:", err)
			return
		}
	}

	screenshots = append(screenshots, fileURL)
	screenshotsJSON, err := json.Marshal(screenshots)
	if err != nil {
		log.Println("Ошибка при кодировании скриншотов:", err)
		return
	}

	_, err = h.DB.ExecContext(ctx, "UPDATE user_tasks SET screenshots = $1 WHERE user_id=$2 AND task_id=$3", screenshotsJSON, userID, taskID)
	if err != nil {
		log.Println("Ошибка при обновлении скриншотов:", err)
		return
	}
}
