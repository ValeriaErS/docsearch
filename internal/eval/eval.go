package eval

import (
	"bufio"
	"context"
	"docsearch/internal/config"
	"docsearch/internal/embed"
	"docsearch/internal/rag"
	"docsearch/internal/retrieve"
	"docsearch/internal/safety"
	"docsearch/internal/vector"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type EvalQuestion struct {
	Query        string   `json:"query"`
	ExpectedDocs []string `json:"expected_docs"`
}

type EvalResult struct {
	Query        string   `json:"query"`
	FoundDocs    []string `json:"found_docs"`
	ExpectedDocs []string `json:"expected_docs"`
	Recall       float64  `json:"recall"`
	Success      bool     `json:"success"`
}

func RunEval(cfg *config.Config, datasetPath string, vectorClient vector.VectorStore) {
	fmt.Println("Запуск")
	if datasetPath == "" {
		datasetPath = "testdata/control/questions.jsonl"
	}

	if cfg.LLM.Provider == "mock" {
		fmt.Println("Внимание: eval запущен в mock-режиме")
		fmt.Println("Результаты могут не отражать реальное качество поиска")
	}

	userForEval := "" // определяю пользователя
	for i, arg := range os.Args {
		if arg == "--user" && i+1 < len(os.Args) {
			userForEval = os.Args[i+1]
			break
		}
	}
	if userForEval == "" {
		fmt.Println("Ошибка: не указан пользователь")
		fmt.Println("Используйте: .\\docsearch.exe eval --user Имя")
		fmt.Println("Пример: .\\docsearch.exe eval --user Екатерина")
		return
	}
	safeUser, err := safety.SanitizeAndValidateUser(userForEval)
	if err != nil {
		fmt.Printf("Ошибка: неверное имя пользователя: %v\n", err)
		return
	}
	userForEval = safeUser

	userDir := "docs/" + userForEval // проверка существует ли папка пользователя
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		fmt.Printf("Ошибка: пользователь %s не существует или нет документов\n", userForEval)
		fmt.Println("Сначала проиндексируйте документы: docsearch.exe index --user", userForEval)
		return
	}

	fmt.Printf("Пользователь: %s\n\n", userForEval)

	file, err := os.Open(datasetPath) //  JSONL
	if err != nil {
		fmt.Println("Ошибка открытия файла:", err)
		return
	}
	defer file.Close()

	var questions []EvalQuestion
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var q EvalQuestion
		if err := json.Unmarshal([]byte(line), &q); err != nil {
			fmt.Printf("Ошибка парсинга строки: %v\n", err)
			continue
		}
		questions = append(questions, q)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	if len(questions) == 0 {
		fmt.Println("Нет вопросов для оценки")
		return
	}

	fmt.Printf("Найдено %d вопросов\n\n", len(questions))

	var results []EvalResult
	totalRecall := 0.0

	for i, q := range questions {
		fmt.Printf("--- Вопрос %d: \"%s\" ---\n", i+1, q.Query)

		texts, docs, scores, answer, pages, _, _, _ := rag.Ask(context.Background(), *cfg, q.Query, userForEval, []map[string]string{}, vectorClient)
		fmt.Printf("Ожидаемые документы: %v\n", q.ExpectedDocs)
		fmt.Printf("Найденные документы: %v\n", docs)
		fmt.Printf("Найдено текстов: %d\n", len(texts))
		fmt.Printf("Страницы: %v\n", pages)

		if len(docs) > 0 {
			fmt.Println("Источники:")
			for i, doc := range docs {
				page := 1
				if i < len(pages) && pages[i] > 0 {
					page = pages[i]
				}
				fmt.Printf("%d. %s (страница %d, оценка: %.2f)\n", i+1, doc, page, scores[i])
			}
		}

		metrics := rag.CountCitations(answer) //считаю ссылки в ответе
		if metrics.TotalCitations > 0 {
			fmt.Printf("Ссылок в ответе: %d\n", metrics.TotalCitations)
			fmt.Printf("Уникальных источников: %d\n", metrics.UniqueSources)
		} else {
			fmt.Println("Ссылок в ответе нет")
		}

		found := 0
		for _, expected := range q.ExpectedDocs {
			for _, foundDoc := range docs {
				if foundDoc == expected {
					found++
					break
				}
			}
		}

		recall := float64(found) / float64(len(q.ExpectedDocs))

		precision := 0.0 //  сколько из найденных документов правильные
		if len(docs) > 0 {
			correct := 0
			for _, foundDoc := range docs {
				for _, expected := range q.ExpectedDocs {
					if foundDoc == expected {
						correct++
						break
					}
				}
			}
			precision = float64(correct) / float64(len(docs))
		}

		f1 := 0.0 // среднее между точностью и полнотой
		if precision+recall > 0 {
			f1 = 2 * precision * recall / (precision + recall)
		}

		totalRecall += recall

		result := EvalResult{
			Query:        q.Query,
			FoundDocs:    docs,
			ExpectedDocs: q.ExpectedDocs,
			Recall:       recall,
			Success:      recall >= 0.5,
		}
		results = append(results, result)

		fmt.Printf("Recall: %.0f%%\n", recall*100)
		fmt.Printf("  Precision: %.0f%%\n", precision*100)
		fmt.Printf("  F1: %.0f%%\n", f1*100)

		if result.Success {
			fmt.Println("успешно")
		} else {
			fmt.Println("не успешно")
		}
		fmt.Println()
	}

	avgRecall := totalRecall / float64(len(questions))
	successes := 0
	for _, r := range results {
		if r.Success {
			successes++
		}
	}

	fmt.Printf("итого:\n")
	fmt.Printf("Средний Recall: %.0f%%\n", avgRecall*100)
	fmt.Printf("Успешных ответов: %d из %d\n", successes, len(questions))

	resultJSON, _ := json.MarshalIndent(results, "", "  ")
	os.WriteFile("eval_results.json", resultJSON, 0644)
	fmt.Println("\n Подробные результаты сохранены в eval_results.json")

	fmt.Println("\n--- Детальные метрики по вопросам ---")
	fmt.Println("№ | Recall | Precision | F1")
	fmt.Println("--|--------|-----------|----")
	for i, r := range results {

		precision := 0.0 // precision для каждого вопроса
		if len(r.FoundDocs) > 0 {
			correct := 0
			for _, foundDoc := range r.FoundDocs {
				for _, expected := range r.ExpectedDocs {
					if foundDoc == expected {
						correct++
						break
					}
				}
			}
			precision = float64(correct) / float64(len(r.FoundDocs))
		}

		f1 := 0.0 // считаю F1
		if precision+r.Recall > 0 {
			f1 = 2 * precision * r.Recall / (precision + r.Recall)
		}
		fmt.Printf("%d | %.0f%% | %.0f%% | %.0f%%\n", i+1, r.Recall*100, precision*100, f1*100)
	}
}

func CompareANNvsExact(cfg *config.Config, userID string, vectorClient vector.VectorStore) {
	fmt.Println("\n Сравнение ANN и Точный поиск")

	file, err := os.Open("testdata/control/questions.jsonl")
	if err != nil {
		fmt.Println("Ошибка загрузки вопросов:", err)
		return
	}
	defer file.Close()

	var questions []EvalQuestion
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var q EvalQuestion
		if err := json.Unmarshal([]byte(line), &q); err != nil {
			fmt.Printf("Ошибка парсинга строки: %v\n", err)
			continue
		}
		questions = append(questions, q)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Ошибка чтения файла:", err)
		return
	}

	allPoints, err := vectorClient.GetAllVectors(context.Background(), vector.CollectionName, userID) // все векторы пользователя из хранилища
	if err != nil {
		fmt.Println("Ошибка получения всех векторов:", err)
		return
	}

	if len(allPoints) == 0 {
		fmt.Println("Нет данных для пользователя", userID)
		fmt.Println("Сначала проиндексируйте документы: docsearch.exe index --user", userID)
		return
	}

	fmt.Printf("Найдено %d чанков для пользователя %s\n\n", len(allPoints), userID)

	for _, q := range questions {
		fmt.Printf("--- Вопрос: %s ---\n", q.Query)

		vec, err := embed.GetEmbedding(context.Background(), q.Query, cfg) //эмбеддинг вопроса
		if err != nil {
			fmt.Println("Ошибка эмбеддинга:", err)
			continue
		}

		queryVec := make([]float64, len(vec))
		for i, v := range vec {
			queryVec[i] = v
		}

		texts := []string{} // подготовка данных для точного поиска
		docs := []string{}
		vectors := [][]float64{}
		for _, point := range allPoints {
			payload, ok := point["payload"].(map[string]interface{})
			if !ok {
				continue
			}
			chunkText, ok := payload["chunk_text"].(string)
			if !ok {
				continue
			}
			docID, ok := payload["doc_id"].(string)
			if !ok {
				continue
			}

			texts = append(texts, chunkText)
			docs = append(docs, docID)

			vecData, ok := point["vector"].([]float32)
			if !ok {
				continue
			}
			vec64 := make([]float64, len(vecData))
			for i, v := range vecData {
				vec64[i] = float64(v)
			}
			vectors = append(vectors, vec64)
		}

		_, exactDocs, exactScores := retrieve.Search(texts, docs, vectors, queryVec, cfg.Retrieval.TopK) // полный перебор

		vec32 := make([]float32, len(vec)) // ANN поиск
		for i, v := range vec {
			vec32[i] = float32(v)
		}
		annResults, err := vectorClient.Search(context.Background(), vector.CollectionName, vec32, cfg.Retrieval.TopK, userID)
		if err != nil {
			fmt.Println("Ошибка ANN поиска:", err)
			continue
		}

		fmt.Printf("Точный поиск:\n")
		if len(exactDocs) > 0 {
			for i, doc := range exactDocs {
				fmt.Printf("%d. %s (оценка: %.4f)\n", i+1, doc, exactScores[i])
			}
		} else {
			fmt.Println("Ничего не найдено")
		}

		fmt.Printf("ANN поиск:\n")
		if len(annResults) > 0 {
			for i, r := range annResults {

				payload, ok := r["payload"].(map[string]interface{})
				if !ok {
					continue
				}
				docID, ok := payload["doc_id"].(string)
				if !ok {
					continue
				}
				fmt.Printf("%d. %s (оценка: %.4f)\n", i+1, docID, r["score"])
			}
		} else {
			fmt.Println("Ничего не найдено")
		}

		if len(exactDocs) > 0 && len(annResults) > 0 {

			annPayload, ok := annResults[0]["payload"].(map[string]interface{})
			if ok {
				annDocID, ok := annPayload["doc_id"].(string)
				if ok && exactDocs[0] == annDocID {
					fmt.Println("ANN и точный поиск дали одинаковый первый результат")
				} else {
					fmt.Printf("Результаты различаются:\n")
					fmt.Printf("ANN: %s\n", annDocID)
					fmt.Printf("Точный: %s\n", exactDocs[0])
				}
			}
		}

		fmt.Println()
	}

	fmt.Println("Вывод ")
	fmt.Println("ANN работает быстрее, но может давать небольшую погрешность")
	fmt.Println("Точный поиск даёт 100% точность, но медленнее")
	fmt.Println("Для больших корпусов рекомендуется использовать ANN")
}
