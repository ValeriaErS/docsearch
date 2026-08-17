package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type Request struct {
	Query string `json:"query"`
}

func main() {
	url := "http://localhost:8080/ask"

	token := os.Getenv("DOCSEARCH_TOKEN")
	if token == "" {
		fmt.Println("Ошибка: переменная DOCSEARCH_TOKEN не установлена")
		fmt.Println("Создай файл .env с содержимым: DOCSEARCH_TOKEN=твой_токен")
		os.Exit(1)
	}

	queries := []string{
		"Что такое RAG?",
		"Как работает поиск?",
		"Сравни RAG и FileAuditor",
		"Что такое эмбеддинг?",
		"Как установить DocSearch?",
		"Что такое векторная база данных?",
		"Как работает реранкинг?",
	}

	concurrency := 10
	totalRequests := 100

	var wg sync.WaitGroup
	results := make(chan time.Duration, totalRequests)
	errors := make(chan error, totalRequests)
	statusCodes := make(map[int]int)
	var mu sync.Mutex

	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			query := queries[idx%len(queries)]
			body, _ := json.Marshal(Request{Query: query})

			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			start := time.Now()
			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Do(req)
			duration := time.Since(start)

			if err != nil {
				errors <- err
				mu.Lock()
				statusCodes[0]++
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			mu.Lock()
			statusCodes[resp.StatusCode]++
			mu.Unlock()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("status: %d", resp.StatusCode)
				return
			}

			results <- duration
		}(i)

		if i%concurrency == 0 && i > 0 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	wg.Wait()
	close(results)
	close(errors)

	totalTime := time.Since(startTime)

	var totalDuration time.Duration
	var minDuration time.Duration = 1<<63 - 1
	var maxDuration time.Duration
	var count int

	for d := range results {
		totalDuration += d
		count++
		if d < minDuration {
			minDuration = d
		}
		if d > maxDuration {
			maxDuration = d
		}
	}

	if count == 0 {
		fmt.Println("============================================================")
		fmt.Println("  ОШИБКА: НЕТ УСПЕШНЫХ ЗАПРОСОВ")
		fmt.Println("============================================================")
		fmt.Println("Проверь:")
		fmt.Println("1. Токен - установи переменную DOCSEARCH_TOKEN")
		fmt.Println("2. Сервер запущен - curl http://localhost:8080/")
		fmt.Println("============================================================")
		fmt.Println("Коды ответов:")
		for code, count := range statusCodes {
			if code == 0 {
				fmt.Printf("  Ошибок: %d\n", count)
			} else {
				fmt.Printf("  %d: %d\n", code, count)
			}
		}
		return
	}

	avgDuration := totalDuration / time.Duration(count)
	rps := float64(count) / totalTime.Seconds()

	fmt.Println("============================================================")
	fmt.Println("  НАГРУЗОЧНОЕ ТЕСТИРОВАНИЕ DOCSEARCH")
	fmt.Println("============================================================")
	fmt.Printf("Всего запросов:   %d\n", count)
	fmt.Printf("Конкурентность:   %d\n", concurrency)
	fmt.Printf("Общее время:      %.2f сек\n", totalTime.Seconds())
	fmt.Printf("RPS:              %.2f\n", rps)
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Среднее время:    %v\n", avgDuration)
	fmt.Printf("Минимальное:      %v\n", minDuration)
	fmt.Printf("Максимальное:     %v\n", maxDuration)
	fmt.Println("------------------------------------------------------------")
	fmt.Println("Коды ответов:")
	for code, count := range statusCodes {
		if code == 0 {
			fmt.Printf("  Ошибок: %d\n", count)
		} else {
			fmt.Printf("  %d: %d\n", code, count)
		}
	}
	fmt.Println("============================================================")

	errorCount := 0
	for range errors {
		errorCount++
	}
	if errorCount > 0 {
		fmt.Printf("Ошибок: %d\n", errorCount)
	} else {
		fmt.Println("Все запросы успешны")
	}
}