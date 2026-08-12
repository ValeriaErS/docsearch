package main

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/eval"
	"docsearch/internal/indexer"
	"docsearch/internal/rag"
	"docsearch/internal/safety"
	"docsearch/internal/server"
	"docsearch/internal/vector"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
	"docsearch/internal/monitor"
	"docsearch/internal/logger"
	
)

type Source struct { // структура для json
	DocID   string  `json:"doc_id"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
	ChunkID string  `json:"chunk_id"`
}

type Response struct {
	Query      string   `json:"query"`
	Answer     string   `json:"answer"`
	Sources    []Source `json:"sources"`
	Model      string   `json:"model"`
	TokensUsed int      `json:"tokens_used"`
	DurationMs int64    `json:"duration_ms"`
}

func runDemo(cfg *config.Config, userID string, vectorClient vector.VectorStore) {
	fmt.Println("Демо режим")
	os.Remove("./.docsearch_index_" + userID + ".json")
	fmt.Println("Удалил старый индекс для демо")

	fmt.Println("Индексирую документы")
	idx := indexer.NewIndexer(cfg, vectorClient, userID)
	err := idx.Index(context.Background())
	if err != nil {
		fmt.Println("Ошибка индексации:", err)
		return
	}
	fmt.Println("Индексация завершена")
	if vc, ok := vectorClient.(*vector.FakeVectorStore); ok {
		fmt.Printf("После индексации: в FakeVectorStore %d точек\n", len(vc.Points))
	}
	outFile := "tmp/demo_result.json"

	questions := []string{
		"What is RAG?",
		"How to install DocSearch?",
		"How to install Linux?",
	}
	allResults := []Response{}

	for _, q := range questions {
		fmt.Printf("\n Вопрос: %s\n", q)
		if vc, ok := vectorClient.(*vector.FakeVectorStore); ok {
			fmt.Printf("Перед поиском: в FakeVectorStore %d точек, ищем для userID=%s\n", len(vc.Points), userID)
		}

		results, docs, scores, answer, _, chunkIDs, tokensUsed, _ := rag.Ask(context.Background(), *cfg, q, userID, []map[string]string{}, vectorClient)
		found := false
		for i := 0; i < len(scores); i++ {
			if scores[i] >= cfg.Retrieval.MinScore {
				found = true
				break
			}
		}

		if !found || answer == "" {
			fmt.Println("Ответ: В документации нет информации по этому вопросу")
			continue
		}
		var sources []Source
		for i := 0; i < len(results); i++ {
			snippet := results[i]
			if len(snippet) > 100 {
				snippet = snippet[:100] + "..."
			}
			sources = append(sources, Source{
				DocID:   docs[i],
				Score:   scores[i],
				Snippet: snippet,
				ChunkID: chunkIDs[i],
			})
		}

		resp := Response{
			Query:      q,
			Answer:     answer,
			Sources:    sources,
			Model:      cfg.LLM.Model,
			TokensUsed: tokensUsed,
			DurationMs: 0,
		}
		allResults = append(allResults, resp)

		fmt.Printf("Ответ: %s\n", answer)
		fmt.Printf("Источников: %d\n", len(docs))
		fmt.Printf("Токенов: %d\n", tokensUsed)
	}
	os.MkdirAll("tmp", 0755)
	jsonData, _ := json.MarshalIndent(allResults, "", "  ")
	os.WriteFile(outFile, jsonData, 0644)
	fmt.Printf("\n Результаты сохранены в %s\n", outFile)
	fmt.Println("\n Демо все")
}

func printHelp() {
	fmt.Println("DocSearch — поиск по документации с RAG")
	fmt.Println()
	fmt.Println("Команды:")
	fmt.Println("ask --query \"вопрос\" --user имя Поиск ответа в документации")
	fmt.Println("serve --addr :8080  Запуск веб-сервера")
	fmt.Println("index --user имя  Индексация документов")
	fmt.Println("eval --user имя  Оценка качества")
	fmt.Println("compare --user имя  Сравнение ANN и точного поиска")
	fmt.Println("demo  Демонстрационный режим")
	fmt.Println("--version Версия программы")
	fmt.Println("analyze Анализ логов пайплайна")
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]

	switch command {
	case "ask":
		askCmd()
	case "serve":
		serveCmd()
	case "web":
		fmt.Println("Команда web устарела. Используйте serve.")
		serveCmd()
	case "index":
		indexCmd()
	case "eval":
		evalCmd()
	case "compare":
		compareCmd()
	case "demo":
		demoCmd()
	case "analyze":
		analyzeCmd()
	case "--version", "-v":
		fmt.Println("DocSearch version 1.0.0")
	default:
		fmt.Printf("Неизвестная команда: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}
func analyzeCmd() {
	analyzeFlag := flag.NewFlagSet("analyze", flag.ExitOnError)
	logFile := analyzeFlag.String("file", "pipeline_logs.jsonl", "Путь к файлу логов")
	topN := analyzeFlag.Int("top", 5, "Количество самых медленных запросов")

	analyzeFlag.Parse(os.Args[2:])

	analyzer, err := logger.NewLogAnalyzer(*logFile)
	if err != nil {
		fmt.Printf("Ошибка чтения логов: %v\n", err)
		return
	}

	analyzer.Summary()
	analyzer.PrintSlowest(*topN)
}

func askCmd() {
	askFlag := flag.NewFlagSet("ask", flag.ExitOnError)
	query := askFlag.String("query", "", "Вопрос к документации")
	userID := askFlag.String("user", "", "Имя пользователя")
	configFile := askFlag.String("config", "configs/config.yml", "Путь к конфигу")
	outFile := askFlag.String("out", "", "Файл для сохранения результата")

	askFlag.Parse(os.Args[2:])

	if *query == "" {
		fmt.Println("Ошибка: требуется --query")
		askFlag.Usage()
		os.Exit(2)
	}
	if *userID == "" {
		fmt.Println("Ошибка: требуется --user")
		askFlag.Usage()
		os.Exit(2)
	}

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Println("Ошибка загрузки конфига:", err)
		return
	}

	safeUser, err := safety.SanitizeAndValidateUser(*userID) // проверка имени пользователя
	if err != nil {
		fmt.Println("Ошибка: неверное имя пользователя:", err)
		return
	}

	vectorClient, err := createVectorClient(cfg)
	if err != nil {
		fmt.Println("Ошибка подключения к векторной БД:", err)
		return
	}

	startTime := time.Now()

	results, docs, scores, answer, _, chunkIDs, tokensUsed, _ := rag.Ask(
		context.Background(),
		*cfg,
		*query,
		safeUser,
		[]map[string]string{},
		vectorClient,
	)

	found := false // проверяю порог
	for i := 0; i < len(scores); i++ {
		if scores[i] >= cfg.Retrieval.MinScore {
			found = true
			break
		}
	}

	if !found { // возврат JSON с пустым sources
		resp := Response{
			Query:      *query,
			Answer:     "В документации нет информации по этому вопросу",
			Sources:    []Source{},
			Model:      cfg.LLM.Model,
			TokensUsed: 0,
			DurationMs: 0,
		}
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonData))
		return
	}

	var sources []Source // собираю источники
	for i := 0; i < len(results); i++ {
		snippet := results[i]
		if len(snippet) > 100 {
			snippet = snippet[:100] + "..."
		}
		sources = append(sources, Source{
			DocID:   docs[i],
			Score:   scores[i],
			Snippet: snippet,
			ChunkID: chunkIDs[i],
		})
	}

	duration := time.Since(startTime).Milliseconds()

	resp := Response{ // собираю ответ
		Query:      *query,
		Answer:     answer,
		Sources:    sources,
		Model:      cfg.LLM.Model,
		TokensUsed: tokensUsed,
		DurationMs: duration,
	}

	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Println("Ошибка формирования:", err)
		return
	}

	if *outFile != "" {
		err := os.WriteFile(*outFile, jsonData, 0644)
		if err != nil {
			fmt.Println("Ошибка сохранения в файл:", err)
		} else {
			fmt.Println("Результат сохранён в", *outFile)
		}
	} else {
		fmt.Println(string(jsonData))
	}
}

func serveCmd() {
	serveFlag := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := serveFlag.String("addr", ":8080", "Адрес для сервера")
	configFile := serveFlag.String("config", "configs/config.yml", "Путь к конфигу")

	metricsFlag := serveFlag.Bool("metrics", false, "Показать метрики и выйти")

	serveFlag.Parse(os.Args[2:])

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Println("Ошибка загрузки конфига:", err)
		return
	}
	if *metricsFlag {
		collector := monitor.GetCollector()
		collector.ExportSummary()
	
	if err := collector.GenerateCharts("metrics"); err != nil {
			fmt.Printf("Ошибка генерации графиков: %v\n", err)
		}
		fmt.Println("\n Открой metrics/dashboard.html для просмотра графиков")
		return
	}

	vectorClient, err := createVectorClient(cfg)
	if err != nil {
		fmt.Println("Ошибка подключения к векторной БД:", err)
		return
	}

	server.RunWeb(cfg, *addr, vectorClient)
}

func indexCmd() {
	indexFlag := flag.NewFlagSet("index", flag.ExitOnError)
	userID := indexFlag.String("user", "", "Имя пользователя")
	configFile := indexFlag.String("config", "configs/config.yml", "Путь к конфигу")

	indexFlag.Parse(os.Args[2:])

	if *userID == "" {
		fmt.Println("Ошибка: требуется --user")
		indexFlag.Usage()
		os.Exit(2)
	}

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Println("Ошибка загрузки конфига:", err)
		return
	}

	safeUser, err := safety.SanitizeAndValidateUser(*userID) // проверка имени пользователя
	if err != nil {
		fmt.Println("Ошибка: неверное имя пользователя:", err)
		return
	}

	vectorClient, err := createVectorClient(cfg)
	if err != nil {
		fmt.Println("Ошибка подключения к векторной БД:", err)
		return
	}

	idx := indexer.NewIndexer(cfg, vectorClient, safeUser)
	err = idx.Index(context.Background())
	if err != nil {
		fmt.Println("Ошибка индексации:", err)
		return
	}

	fmt.Println("Индексация завершена")
}

func evalCmd() {
	evalFlag := flag.NewFlagSet("eval", flag.ExitOnError)
	userID := evalFlag.String("user", "", "Имя пользователя")
	configFile := evalFlag.String("config", "configs/config.yml", "Путь к конфигу")
	datasetPath := evalFlag.String("dataset", "testdata/control/questions.jsonl", "Путь к JSONL с вопросами")

	evalFlag.Parse(os.Args[2:])

	if *userID == "" {
		fmt.Println("Ошибка: требуется --user")
		evalFlag.Usage()
		os.Exit(2)
	}

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Println("Ошибка загрузки конфига:", err)
		return
	}

	if _, err := safety.SanitizeAndValidateUser(*userID); err != nil {  // проверка имени пользователя
		fmt.Println("Ошибка: неверное имя пользователя:", err)
		return
	}

	vectorClient, err := createVectorClient(cfg)
	if err != nil {
		fmt.Println("Ошибка подключения к векторной БД:", err)
		return
	}

	eval.RunEval(cfg, *datasetPath, vectorClient)
}
func compareCmd() {
	compareFlag := flag.NewFlagSet("compare", flag.ExitOnError)
	userID := compareFlag.String("user", "demo", "Имя пользователя")
	configFile := compareFlag.String("config", "configs/config.yml", "Путь к конфигу")

	compareFlag.Parse(os.Args[2:])

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Println("Ошибка загрузки конфига:", err)
		return
	}

	safeUser, err := safety.SanitizeAndValidateUser(*userID)
	if err != nil {
		fmt.Println("Ошибка: неверное имя пользователя:", err)
		return
	}

	vectorClient, err := createVectorClient(cfg)
	if err != nil {
		fmt.Println("Ошибка подключения к векторной БД:", err)
		return
	}

	eval.CompareANNvsExact(cfg, safeUser, vectorClient)
}

func demoCmd() {
	demoFlag := flag.NewFlagSet("demo", flag.ExitOnError)
	configFile := demoFlag.String("config", "configs/config.yml", "Путь к конфигу")
	demoFlag.Parse(os.Args[2:])

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Println("Ошибка загрузки конфига:", err)
		return
	}
	fakeClient := vector.NewFakeVectorStore()
	fmt.Println("Использую FakeVectorStore (mock-режим)")
	runDemo(cfg, "demo", fakeClient)
}

func createVectorClient(cfg *config.Config) (vector.VectorStore, error) {
	if cfg.Embeddings.Provider == "mock" {
		return vector.NewFakeVectorStore(), nil
	}

	client, err := vector.NewQdrantClient()
	if err != nil {
		return nil, err
	}
	if err := client.Ping(context.Background()); err != nil {
		return nil, err
	}
	client.VectorSize = cfg.Embeddings.VectorSize
	return client, nil

	
}