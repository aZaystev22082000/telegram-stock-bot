package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql" // ДОБАВЛЕНО: импорт MySQL драйвера
)

// Структуры для Alpha Vantage API
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

type APIErrorResponse struct {
	ErrorMessage string `json:"Error Message"`
	Information  string `json:"Information"`
	Note         string `json:"Note"`
}

// ДОБАВЛЕНО: глобальная переменная для подключения к БД
var db *sql.DB

// ДОБАВЛЕНО: функция инициализации базы данных
func initDB() {
	var err error
	dbSource := "root:Froliner564@tcp(localhost:3306)/sys"

	db, err = sql.Open("mysql", dbSource)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Ошибка ping базы данных: %v", err)
	}

	log.Println("✅ Успешное подключение к базе данных")
}

// Существующие функции для работы с API
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

func getStockPrice(symbol, apiKey string, maxRetries int) (string, error) {
	client := createHTTPClient()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Попытка %d/%d для тикера %s\n", attempt, maxRetries, symbol)

		price, err := fetchStockPrice(client, symbol, apiKey)
		if err == nil {
			return price, nil
		}

		fmt.Printf("Ошибка (попытка %d): %v\n", attempt, err)

		if attempt < maxRetries {
			backoff := time.Duration(attempt) * 2 * time.Second
			fmt.Printf("Ждем %v перед повторной попыткой...\n", backoff)
			time.Sleep(backoff)
		}
	}

	return "", fmt.Errorf("не удалось получить цену после %d попыток", maxRetries)
}

func fetchStockPrice(client *http.Client, symbol, apiKey string) (string, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", symbol, apiKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var quoteResponse GlobalQuoteResponse
	if err := json.Unmarshal(body, &quoteResponse); err != nil {
		return "", fmt.Errorf("ошибка разбора JSON: %v\nТело ответа: %s", err, string(body))
	}

	if quoteResponse.GlobalQuote.Price == "" {
		return "", fmt.Errorf("цена не найдена в ответе API\nПолный ответ: %s", string(body))
	}

	return quoteResponse.GlobalQuote.Price, nil
}

// ДОБАВЛЕНО: функция для тестирования нескольких тикеров
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

		time.Sleep(1 * time.Second)
	}
}

func main() {
	// ДОБАВЛЕНО: инициализация базы данных
	initDB()
	runBot()
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
