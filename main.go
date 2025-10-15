package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Структура для парсинга ответа от GLOBAL_QUOTE
type GlobalQuoteResponse struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Open             string `json:"02. open"`
		High             string `json:"03. high"`
		Low              string `json:"04. low"`
		Price            string `json:"05. price"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
		PreviousClose    string `json:"08. previous close"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
	} `json:"Global Quote"`
}

// Структура для ошибок API
type APIErrorResponse struct {
	ErrorMessage string `json:"Error Message"`
	Information  string `json:"Information"`
	Note         string `json:"Note"`
}

// Создание HTTP-клиента с таймаутами
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// Получение цены акции с повторами при ошибках
func getStockPrice(symbol, apiKey string, maxRetries int) (string, error) {
	client := createHTTPClient()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Попытка %d/%d для тикера %s\n", attempt, maxRetries, symbol)

		price, err := fetchStockPrice(client, symbol, apiKey)
		if err == nil {
			return price, nil
		}

		fmt.Printf("Ошибка (попытка %d): %v\n", attempt, err)

		// Если это не последняя попытка, ждем перед повторением
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * 2 * time.Second
			fmt.Printf("Ждем %v перед повторной попыткой...\n", backoff)
			time.Sleep(backoff)
		}
	}

	return "", fmt.Errorf("не удалось получить цену после %d попыток", maxRetries)
}

// Основная функция получения данных от API
func fetchStockPrice(client *http.Client, symbol, apiKey string) (string, error) {
	// Формируем URL для GLOBAL_QUOTE
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", symbol, apiKey)

	// Создаем запрос
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	// Сначала проверяем на ошибки API
	//var apiError APIErrorResponse
	//if err := json.Unmarshal(body, &apiError); err == nil {
	//if apiError.ErrorMessage != "" {
	//return "", fmt.Errorf("ошибка API: %s", apiError.ErrorMessage)
	//}
	//if apiError.Information != "" {
	//return "", fmt.Errorf("информация от API: %s", apiError.Information)
	//}
	//if apiError.Note != "" {
	//return "", fmt.Errorf("примечание от API: %s", apiError.Note)
	//}
	//}

	// Парсим основной ответ
	var quoteResponse GlobalQuoteResponse
	if err := json.Unmarshal(body, &quoteResponse); err != nil {
		return "", fmt.Errorf("ошибка разбора JSON: %v\nТело ответа: %s", err, string(body))
	}

	// Проверяем, есть ли цена в ответе
	if quoteResponse.GlobalQuote.Price == "" {
		return "", fmt.Errorf("цена не найдена в ответе API\nПолный ответ: %s", string(body))
	}

	return quoteResponse.GlobalQuote.Price, nil
}

// Функция для тестирования нескольких тикеров
func testMultipleTickers(apiKey string) {
	tickers := []string{
		"SBER.ME", // Сбербанк (Московская биржа)
		"GAZP.ME", // Газпром
		"IBM",     // IBM (для теста - обычно работает)
		"AAPL",    // Apple
	}

	for _, ticker := range tickers {
		fmt.Printf("\n=== Запрос для %s ===\n", ticker)

		price, err := getStockPrice(ticker, apiKey, 3)
		if err != nil {
			fmt.Printf("Ошибка для %s: %v\n", ticker, err)
		} else {
			fmt.Printf("Текущая цена %s: %s\n", ticker, price)
		}

		// Пауза между запросами чтобы не превысить лимиты
		time.Sleep(1 * time.Second)
	}
}

func main() {
	apiKey := "Z33Q2SGS87R4NCV9"

	fmt.Println("=== Получение котировок акций ===")

	// Запрашиваем у пользователя ввод тикера
	var symbol string
	fmt.Print("Введите тикер акции (например, SBER.ME): ")
	_, err := fmt.Scanln(&symbol)
	if err != nil {
		fmt.Printf("Ошибка ввода: %v\n", err)
		return
	}

	price, err := getStockPrice(symbol, apiKey, 3)
	if err != nil {
		fmt.Printf("Произошла ошибка: %v\n", err)
	} else {
		fmt.Printf("Текущая цена акции %s: %s\n", symbol, price)
	}
}
