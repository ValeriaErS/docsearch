package query

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"fmt"
	"strings"
)

func CompressChunks(ctx context.Context, chunks []string, question string, cfg *config.Config) ([]string, error) {  // сжимаю каждый чанк до 2-3 ключевых предложений для экономии токенов и улучшения ответов
	if cfg.LLM.Provider == "mock" || len(chunks) == 0 {
		return chunks, nil
	}

	fmt.Printf("Сжимаю %d чанков...\n", len(chunks))

	batchSize := 4
	compressed := []string{}

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[i:end]

		
		prompt := fmt.Sprintf(`Ты — помощник по сжатию текста.
Сожми КАЖДЫЙ из следующих фрагментов до 2-3 предложений.
Оставь только самое важное для ответа на вопрос: "%s"

Фрагменты:
%s

Выдай сжатые версии в том же порядке, разделяя их тремя дефисами: ---

Пример:
Фрагмент 1: "RAG - это технология. Она помогает искать информацию. Используется в чат-ботах."
Фрагмент 2: "Документы хранятся в векторной базе. Поиск идет по смыслу."

Сжато:
RAG помогает искать информацию в чат-ботах.
---
Документы хранятся в векторной базе для поиска по смыслу.`, question, strings.Join(batch, "\n---\n"))

		messages := []map[string]string{
			{"role": "system", "content": "Ты помощник по сжатию текста. Оставляй только суть."},
			{"role": "user", "content": prompt},
		}

		response, _, err := llm.GetAnswerWithHistory(ctx, "", []string{}, []string{}, []int{}, messages, cfg)
		if err != nil {
			fmt.Printf("Ошибка сжатия: %v, пропускаю\n", err)
			compressed = append(compressed, batch...)
			continue
		}

		parts := strings.Split(response, "---")   // разбираю ответ на отдельные чанки
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) > 10 {
				compressed = append(compressed, part)
			}
		}
	}

	fmt.Printf("Сжато: %d → %d чанков\n", len(chunks), len(compressed))
	return compressed, nil
}