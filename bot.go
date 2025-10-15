package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Структуры для парсинга ответа от Alpha Vantage
type GlobalQuoteResponse struct {
	GlobalQuote struct {
		Symbol string `json:"01. symbol"`
		Price  string `json:"05. price"`
	} `json:"Global Quote"`
}

type APIErrorResponse struct {
	ErrorMessage string `json:"Error Message"`
	Information  string `json:"Information"`
	Note         string `json:"Note"`
}

func main() {
	// ЗАМЕНИТЕ "ВАШ_ТОКЕН_OT_BOTFATHER" на реальный токен!
	bot, err := tgbotapi.NewBotAPI("8167489635:AAFPlmh2Y--sETZaz-josSMVxMDU87PQqzU")
	if err != nil {
		log.Panic("Ошибка инициализации бота: ", err)
	}

	bot.Debug = true
	log.Printf("Авторизация успешна! Бот %s запущен", bot.Self.UserName)

	// Получаем последний update_id для начала
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Создаем канал для получения обновлений
	updates := bot.GetUpdatesChan(u)

	// Обрабатываем входящие сообщения
	for update := range updates {
		// Игнорируем любые сообщения без текста
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		// Создаем сообщение для ответа
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		// Обрабатываем команду /start
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg.Text = "Привет! Я бот для проверки цен акций.\nПросто отправь мне тикер (например, SBER.ME или AAPL), и я найду его текущую стоимость."
			default:
				msg.Text = "Неизвестная команда. Просто отправь мне тикер акции."
			}
		} else {
			// Обрабатываем обычное сообщение как тикер
			ticker := strings.TrimSpace(update.Message.Text)
			if ticker == "" {
				msg.Text = "Пожалуйста, укажи тикер. Например: SBER.ME"
			} else {
				// Получаем цену акции
				price, err := getStockPrice(ticker, "Z33Q2SGS87R4NCV9") // Ваш API-ключ Alpha Vantage
				if err != nil {
					msg.Text = fmt.Sprintf("Произошла ошибка: %v", err)
				} else {
					msg.Text = fmt.Sprintf("Текущая цена акции %s: %s", ticker, price)
				}
			}
		}

		// Отправляем сообщение пользователю
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
	}
}

// Функция для получения цены акции
func getStockPrice(symbol, apiKey string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", symbol, apiKey)

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	// Сначала проверяем на ошибки API
	var apiError APIErrorResponse
	if err := json.Unmarshal(body, &apiError); err == nil {
		if apiError.ErrorMessage != "" {
			return "", fmt.Errorf("ошибка API: %s", apiError.ErrorMessage)
		}
		if apiError.Information != "" {
			return "", fmt.Errorf("информация от API: %s", apiError.Information)
		}
		if apiError.Note != "" {
			return "", fmt.Errorf("примечание от API: %s", apiError.Note)
		}
	}

	var data GlobalQuoteResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("ошибка разбора JSON: %v", err)
	}

	if data.GlobalQuote.Price == "" {
		return "", fmt.Errorf("не удалось получить цену для тикера %s", symbol)
	}

	return data.GlobalQuote.Price, nil
}

// Альтернативная функция для ручного управления offset (если автоматический не работает)
func runBotWithManualOffset(bot *tgbotapi.BotAPI) {
	offset := 0
	for {
		u := tgbotapi.NewUpdate(offset)
		u.Timeout = 60

		updates, err := bot.GetUpdates(u)
		if err != nil {
			log.Printf("Ошибка получения обновлений: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.Message == nil {
				continue
			}

			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					msg.Text = "Привет! Я бот для проверки цен акций."
				default:
					msg.Text = "Неизвестная команда"
				}
			} else {
				ticker := strings.TrimSpace(update.Message.Text)
				price, err := getStockPrice(ticker, "Z33Q2SGS87R4NCV9")
				if err != nil {
					msg.Text = fmt.Sprintf("Ошибка: %v", err)
				} else {
					msg.Text = fmt.Sprintf("Цена %s: %s", ticker, price)
				}
			}

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки: %v", err)
			}
		}

		// Небольшая пауза между запросами
		time.Sleep(1 * time.Second)
	}
}
