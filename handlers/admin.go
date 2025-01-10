// handlers/admin.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"telegram_bot/models"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleAdminAddTask(ctx context.Context, update tgbotapi.Update) {
	// Предложение выбрать тип задания
	userID := update.Message.From.ID
	msgKeyboard := h.AdminMenuTask
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите тип задания для добавления:")
	msg.ReplyMarkup = msgKeyboard
	h.Bot.Send(msg)

	// Установка состояния администратора
	err := h.DB.SetUserState(ctx, userID, string(models.StateAwaitingTaskCategory))
	if err != nil {
		log.Printf("Ошибка при установке состояния: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при установке состояния.")
		h.Bot.Send(msg)
	}
}

func (h *Handler) HandleAdminTaskCategorySelection(ctx context.Context, update tgbotapi.Update) {
	categoryText := update.Message.Text
	adminID := update.Message.From.ID

	if categoryText == "Отменить добавление" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Добавление задания отменено.")
		msg.ReplyMarkup = h.AdminMenuTask
		h.Bot.Send(msg)
		h.DB.SetUserState(ctx, adminID, string(models.StateNone))
		return
	}

	// Валидация выбранной категории
	var selectedCategory models.Category
	switch categoryText {
	case string(models.CategoryAvito):
		selectedCategory = models.CategoryAvito
	case string(models.CategoryYandex):
		selectedCategory = models.CategoryYandex
	case string(models.CategoryGoogle):
		selectedCategory = models.CategoryGoogle
	case string(models.Category2GIS):
		selectedCategory = models.Category2GIS
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверная категория. Пожалуйста, выберите одну из доступных.")
		msg.ReplyMarkup = h.AdminMenuTask // Или клавиатура с категориями
		h.Bot.Send(msg)
		return
	}
	// Сохранение выбранной категории
	err := h.DB.SetTempData(ctx, update.Message.From.ID, "new_task_category", selectedCategory)
	if err != nil {
		// Обработка ошибки
		log.Printf("Ошибка при сохранении временных данных: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при сохранении категории задания.")
		h.Bot.Send(msg)
		return
	}

	// Запрос описания задания
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите описание задания:")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	h.Bot.Send(msg)

	// Обновление состояния администратора
	h.DB.SetUserState(ctx, update.Message.From.ID, string(models.StateAwaitingTaskDescription))
}

func (h *Handler) HandleAdminTaskDescription(ctx context.Context, update tgbotapi.Update) {
	description := update.Message.Text
	adminID := update.Message.From.ID

	// Получение сохраненной категории
	tempData, err := h.DB.GetTempData(ctx, adminID, "new_task_category")
	if err != nil {
		log.Printf("Ошибка при получении временных данных: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при получении категории задания.")
		h.Bot.Send(msg)
		return
	}

	// Приведение типа interface{} к string
	selectedCategory, ok := tempData.(models.Category)
	if !ok {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка: Некорректный тип категории.")
		if _, err := h.Bot.Send(msg); err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
		return
	}

	// Создание задания в базе данных
	task := models.Task{
		Category:         selectedCategory,
		Description:      description,
		Link:             "https://example.com",
		IsActive:         true,
		CreatedAt:        time.Now(),
		Status:           "New",
		ScreenshotFileID: "file_id_12345",
	}

	// Сохранение задания в базе данных
	err = h.DB.CreateTask(ctx, &task)
	if err != nil {
		log.Printf("Ошибка при создании задачи: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при создании задачи.")
		h.Bot.Send(msg)
		return
	}

	// Уведомление об успешном добавлении задания
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Задание успешно добавлено!")
	msg.ReplyMarkup = h.AdminMenuTask
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
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при получении заданий для проверки."))
		return
	}

	if len(tasks) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нет заданий для проверки.")
		h.Bot.Send(msg)
		return
	}

	for _, task := range tasks {
		// Формирование информации о задании
		taskInfo := fmt.Sprintf(
			"👤 *Пользователь ID:* %d\n"+
				"📂 *Категория:* %s\n"+
				"📄 *Задание:* %s\n"+
				"📝 *Описание:* %s\n"+
				"🔗 *Ссылка:* %s\n"+
				"📅 *Создано:* %s\n",
			task.UserID,
			task.Category,
			task.ID, // Если ID задания необходимо отображать
			task.Description,
			task.Link,
			task.CreatedAt.Format("2006-01-02 15:04:05"),
		)
		// Создание кнопок для одобрения и отклонения
		approveButton := tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve_%d", task.ID))
		rejectButton := tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", task.ID))
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(approveButton, rejectButton),
		)

		// Создание сообщения с фотографией
		photoMsg := tgbotapi.NewPhoto(
			update.Message.Chat.ID,
			tgbotapi.FileID(task.ScreenshotFileID), // Используем file_id для отправки фото
		)
		photoMsg.Caption = taskInfo
		photoMsg.ParseMode = "Markdown"
		photoMsg.ReplyMarkup = keyboard

		// Отправка сообщения
		if _, err := h.Bot.Send(photoMsg); err != nil {
			// Логирование ошибки, если отправка не удалась
			fmt.Printf("Ошибка при отправке фото для задания ID %d: %v\n", task.ID, err)
		}
	}
}
