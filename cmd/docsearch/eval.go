package main

import (
    "encoding/json"
    "fmt"
    "os"
    "docsearch/internal/config"
    "docsearch/internal/rag"
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

func runEval(cfg *config.Config) {
    fmt.Println("\n Запуск")

    if cfg.LLM.Provider == "mock" {
        fmt.Println("Внимание: eval запущен в mock-режиме")
        fmt.Println(" Результаты могут не отражать реальное качество поиска\n")
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

        texts, docs, _, _, _, _ := rag.Ask(*cfg, q.Query, userForEval)

        fmt.Printf("Ожидаемые документы: %v\n", q.ExpectedDocs)
        fmt.Printf("Найденные документы: %v\n", docs)
        fmt.Printf("Найдено текстов: %d\n", len(texts))

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
        totalRecall += recall

        result := EvalResult{
            Query: q.Query,
            FoundDocs: docs,
            ExpectedDocs: q.ExpectedDocs,
            Recall: recall,
            Success: recall >= 0.5,
        }
        results = append(results, result)

        fmt.Printf("  Recall: %.0f%%\n", recall*100)
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
}