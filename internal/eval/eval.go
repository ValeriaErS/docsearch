package eval

import (
    "encoding/json"
    "fmt"
    "os"
    "docsearch/internal/config"
    "docsearch/internal/rag"
    "context"
    
    
)

type EvalQuestion struct {
    Query string  `json:"query"`
    ExpectedDocs []string `json:"expected_docs"`
}

type EvalResult struct {
    Query string `json:"query"`
    FoundDocs []string `json:"found_docs"`
    ExpectedDocs []string `json:"expected_docs"`
    Recall float64 `json:"recall"`
    Success bool `json:"success"`
}

func RunEval(cfg *config.Config) {
    fmt.Println("Запуск")

    if cfg.LLM.Provider == "mock" {
        fmt.Println("Внимание: eval запущен в mock-режиме")
        fmt.Println("Результаты могут не отражать реальное качество поиска")
    }

    userForEval := ""  // определяю пользователя
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
userDir := "docs/" + userForEval  // проверка существует ли папка пользователя
if _, err := os.Stat(userDir); os.IsNotExist(err) {
    fmt.Printf("Ошибка: пользователь %s не существует или нет документов\n", userForEval)
    fmt.Println("Сначала проиндексируйте документы: docsearch.exe index --user", userForEval)
    return
}

    fmt.Printf("Пользователь: %s\n\n", userForEval)

    data, err := os.ReadFile("testdata/eval.json")
    if err != nil {
        fmt.Println("Файл testdat/eval.json не найден")
        return
    }

    var questions []EvalQuestion
    err = json.Unmarshal(data, &questions)
    if err != nil {
        fmt.Println("Ошибка чтения eval.json:", err)
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

      texts, docs, scores, answer, pages, _, _ := rag.Ask(context.Background(), *cfg, q.Query, userForEval, []map[string]string{}, nil)
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

        metrics:=rag.CountCitations(answer) //считаю ссылки в ответе
        if metrics.TotalCitations>0{
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
        
        precision := 0.0  //  сколько из найденных документов правильные
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

f1 := 0.0   // среднее между точностью и полнотой
if precision+recall > 0 {
    f1 = 2 * precision * recall / (precision + recall)
}

        totalRecall += recall

        result := EvalResult{
            Query: q.Query,
            FoundDocs: docs,
            ExpectedDocs: q.ExpectedDocs,
            Recall: recall,
            Success: recall >= 0.5,
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
    
    precision := 0.0  // precision для каждого вопроса
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
    
    f1 := 0.0   // считаю F1
    if precision+r.Recall > 0 {
        f1 = 2 * precision * r.Recall / (precision + r.Recall)
    }
    fmt.Printf("%d | %.0f%% | %.0f%% | %.0f%%\n", i+1, r.Recall*100, precision*100, f1*100)
}
}