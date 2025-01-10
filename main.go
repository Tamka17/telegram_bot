// main.go
package main

import (
	"context"
	"log"
	"os"
	"strings"

	"telegram_bot/database"
	"telegram_bot/handlers"
	"telegram_bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	// Загрузка переменных окружения
	err := godotenv.Load()
	if err != nil {
		log.Printf(".env файл не найден, продолжаем с системными переменными")
	}

	// Инициализация БД
	database.InitDB()
	defer database.CloseDB()

	// Получение токена из переменных окружения
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан в переменных окружения")
	}

	// Инициализация Telegram бота
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Авторизовался на аккаунте %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	Database := database.NewDatabase()

	// Определение администраторов (можно извлекать из базы данных)

	handler := handlers.Handler{
		DB:  Database,
		Bot: bot,
	}

	for update := range updates {
		if update.Message != nil {
			ctx := context.Background()
			userID := update.Message.From.ID
			var user models.User

			// Проверяем состояние пользователя
			userState, _ := handler.DB.GetUserState(ctx, userID)
			if userState == "awaiting_card_number" {
				handler.HandleCardNumberReceived(ctx, update)
				continue
			}

			// Добавьте другие состояния по необходимости

			// Обработка команд
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handler.Start(ctx, update)
				default:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда.")
					msg.ReplyMarkup = handler.AdminMenu
					handler.Bot.Send(msg)
				}
				continue
			}

			// Обработка обычных текстовых сообщений
			chatID := update.Message.Chat.ID
			telegramID := update.Message.From.ID
			text := update.Message.Text

			// Проверка, является ли пользователь администратором
			if user.Admin {
				switch text {
				case "Добавить задание":
					handler.HandleAdminAddTask(ctx, update)
				case "Проверить задания":
					handler.HandleAdminCheckTasks(ctx, update)
				case "Главное меню":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Главное меню:")
					msg.ReplyMarkup = handler.AdminMenu
					handler.Bot.Send(msg)
				default:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда. Пожалуйста, выберите действие из меню.")
					msg.ReplyMarkup = handler.AdminMenu
					handler.Bot.Send(msg)
				}
				continue
			}

			// Обработка сообщений обычных пользователей
			switch text {
			case "Показать баланс":
				handler.ShowBalance(ctx, chatID, telegramID)
			case "Личный кабинет":
				handler.HandleShowAccount(ctx, update)
			case "Взять задание":
				handler.HandleAssignTask(ctx, update)
			case "Вывести средства":
				handler.HandleWithdrawRequest(ctx, update)
			case "Обратиться в техподдержку":
				handler.HandleSupport(ctx, update)
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Команда не распознана. Пожалуйста, выберите действие из меню.")
				msg.ReplyMarkup = handler.Keyboard
				handler.Bot.Send(msg)
			}

			if update.CallbackQuery != nil {
				data := update.CallbackQuery.Data
				if strings.HasPrefix(data, "starttask") || strings.HasPrefix(data, "nextstage") {
					handler.HandleTaskAction(context.Background(), update)
				} else if strings.HasPrefix(data, "verify") {
					handler.HandleAdminCheckTasks(context.Background(), update)
				}
			}
		}
	}
}
