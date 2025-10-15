package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ДОБАВЛЕНО: функции для работы с избранным
func AddToFavorites(chatID int64, ticker string) error {
	// 1. Проверяем количество существующих тикеров
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM user_favorites WHERE chat_id = ?", chatID).Scan(&count)
	if err != nil {
		return fmt.Errorf("ошибка проверки количества: %v", err)
	}

	if count >= 5 {
		return fmt.Errorf("нельзя добавить больше 5 тикеров в избранное")
	}

	// 2. Проверяем, существует ли уже такой тикер
	var existingTicker string
	err = db.QueryRow("SELECT ticker FROM user_favorites WHERE chat_id = ? AND ticker = ?", chatID, ticker).Scan(&existingTicker)
	if err == nil {
		return fmt.Errorf("тикер %s уже есть в избранном", ticker)
	}

	// 3. Проверяем, что тикер существует через API
	_, err = getStockPrice(ticker, "Z33Q2SGS87R4NCV9", 1)
	if err != nil {
		return fmt.Errorf("тикер %s не найден на бирже", ticker)
	}

	// 4. Добавляем в базу данных
	_, err = db.Exec("INSERT INTO user_favorites (chat_id, ticker) VALUES (?, ?)", chatID, ticker)
	if err != nil {
		return fmt.Errorf("ошибка добавления в базу: %v", err)
	}

	log.Printf("Добавлен тикер %s для пользователя %d", ticker, chatID)
	return nil
}

func RemoveFromFavorites(chatID int64, ticker string) error {
	result, err := db.Exec("DELETE FROM user_favorites WHERE chat_id = ? AND ticker = ?", chatID, ticker)
	if err != nil {
		return fmt.Errorf("ошибка удаления: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("тикер %s не найден в избранном", ticker)
	}

	log.Printf("Удален тикер %s для пользователя %d", ticker, chatID)
	return nil
}

func GetFavorites(chatID int64) ([]string, error) {
	rows, err := db.Query("SELECT ticker FROM user_favorites WHERE chat_id = ? ORDER BY added_at DESC", chatID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка: %v", err)
	}
	defer rows.Close()

	var favorites []string
	for rows.Next() {
		var ticker string
		if err := rows.Scan(&ticker); err != nil {
			return nil, err
		}
		favorites = append(favorites, ticker)
	}

	return favorites, nil
}

func GetFavoritesWithPrices(chatID int64) (map[string]string, error) {
	tickers, err := GetFavorites(chatID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, ticker := range tickers {
		price, err := getStockPrice(ticker, "Z33Q2SGS87R4NCV9", 1)
		if err != nil {
			result[ticker] = "Ошибка получения цены"
		} else {
			result[ticker] = price
		}

		time.Sleep(300 * time.Millisecond)
	}

	return result, nil
}

func runBot() {
	bot, err := tgbotapi.NewBotAPI("8167489635:AAFPlmh2Y--sETZaz-josSMVxMDU87PQqzU")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		//if update.Message.From.UserName == "sgoreela" {
		//	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🚫 Вы заблокированы, идите нахуй")
		//	if _, err := bot.Send(msg); err != nil {
		//		log.Printf("Ошибка отправки сообщения: %v", err)
		//	}
		//	continue // Прерываем обработку для этого пользователя
		//}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg.Text = "Привет! Я бот для проверки цен акций.\n\n" +
					"Доступные команды:\n" +
					"/add <тикер> - добавить в избранное\n" +
					"/remove <тикер> - удалить из избранного\n" +
					"/list - показать избранное\n" +
					"/prices - цены избранных акций\n\n" +
					"Просто отправь мне тикер (например, SBER.ME или AAPL), и я найду его текущую стоимость."

			// ДОБАВЛЕНО: обработка команды /add
			case "add":
				ticker := strings.TrimSpace(update.Message.CommandArguments())
				if ticker == "" {
					msg.Text = "Укажите тикер для добавления. Например: /add SBER.ME"
				} else {
					err := AddToFavorites(update.Message.Chat.ID, ticker)
					if err != nil {
						msg.Text = fmt.Sprintf("❌ Ошибка: %v", err)
					} else {
						msg.Text = fmt.Sprintf("✅ Тикер %s добавлен в избранное!", ticker)
					}
				}

			// ДОБАВЛЕНО: обработка команды /remove
			case "remove":
				ticker := strings.TrimSpace(update.Message.CommandArguments())
				if ticker == "" {
					msg.Text = "Укажите тикер для удаления. Например: /remove SBER.ME"
				} else {
					err := RemoveFromFavorites(update.Message.Chat.ID, ticker)
					if err != nil {
						msg.Text = fmt.Sprintf("❌ Ошибка: %v", err)
					} else {
						msg.Text = fmt.Sprintf("✅ Тикер %s удален из избранного", ticker)
					}
				}

			// ДОБАВЛЕНО: обработка команды /list
			case "list":
				favorites, err := GetFavorites(update.Message.Chat.ID)
				if err != nil {
					msg.Text = "❌ Ошибка при получении списка избранного"
				} else if len(favorites) == 0 {
					msg.Text = "📭 Ваш список избранного пуст\n\nДобавьте тикеры командой /add <тикер>"
				} else {
					msg.Text = "⭐ Ваши избранные тикеры:\n" + strings.Join(favorites, "\n") +
						"\n\nДля просмотра цен используйте /prices"
				}

			// ДОБАВЛЕНО: обработка команды /prices
			case "prices":
				prices, err := GetFavoritesWithPrices(update.Message.Chat.ID)
				if err != nil {
					msg.Text = "❌ Ошибка при получении цен"
				} else if len(prices) == 0 {
					msg.Text = "📭 Ваш список избранного пуст"
				} else {
					var priceList []string
					for ticker, price := range prices {
						priceList = append(priceList, fmt.Sprintf("%s: %s", ticker, price))
					}
					msg.Text = "📊 Цены ваших избранных акций:\n" + strings.Join(priceList, "\n")
				}

			default:
				msg.Text = "Неизвестная команда. Просто отправь мне тикер акции."
			}
		} else {
			// Обрабатываем обычное сообщение как тикер
			ticker := strings.TrimSpace(update.Message.Text)
			if ticker == "" {
				msg.Text = "Пожалуйста, укажи тикер. Например: SBER.ME"
			} else {
				price, err := getStockPrice(ticker, "Z33Q2SGS87R4NCV9", 3)
				if err != nil {
					msg.Text = fmt.Sprintf("Произошла ошибка: %v", err)
				} else {
					msg.Text = fmt.Sprintf("Текущая цена акции %s: %s", ticker, price)
				}
			}
		}

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
	}
}
