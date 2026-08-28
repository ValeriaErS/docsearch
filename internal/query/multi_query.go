package query

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"fmt"
	"strings"
)

func GenerateMultiQueries(ctx context.Context, query string, cfg *config.Config) ([]string, error) {  //  генерирует несколько вариантов одного вопроса
	fmt.Printf("[MultiQuery] Генерирую варианты запроса: '%s'\n", query)
	if cfg.LLM.Provider == "mock" {
		return []string{query}, nil
	}

	systemPrompt := `Ты — генератор вариантов поисковых запросов.
Придумай 5 разных вариантов одного и того же вопроса.

Правила:
1. Все варианты должны означать одно и то же
2. Используй разные формулировки
3. Варианты должны быть на русском языке
4. Выдай только список вариантов, каждый с новой строки

Пример:
Вопрос: "Как установить DocSearch?"
Варианты:
установка DocSearch
DocSearch установка
как запустить DocSearch
инструкция по установке DocSearch
DocSearch монтаж

Вопрос: %s
Варианты:`

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": fmt.Sprintf("Вопрос: %s", query)},
	}

	response, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, cfg)
	if err != nil {
		return []string{query}, err
	}

	lines := strings.Split(response, "\n") // разбор ответа
	variants := []string{query}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		line = strings.TrimLeft(line, "0123456789. )-")  // убираю цифры и пунктуацию в начале
		line = strings.TrimSpace(line)
		if len(line) > 3 && line != query {
			variants = append(variants, line)
		}
	}

	if len(variants) > 5 {  // оставляю не больше 5 вариантов
		variants = variants[:5]
	}
	fmt.Printf("[MultiQuery] Сгенерировано %d вариантов:\n", len(variants))   
    for i, v := range variants {
        fmt.Printf("Вариант %d: '%s'\n", i+1, v)
    }
	return variants, nil
}