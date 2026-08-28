package query

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"encoding/json"
	"fmt"
	"strings"
)

type DecomposeResult struct { //результат декомпозиции запроса
	Original   string   `json:"original"`
	SubQueries []string `json:"sub_queries"`
	IsComplex  bool     `json:"is_complex"`
	Reason     string   `json:"reason"`
}

func ShouldDecompose(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))

	complexMarkers := []string{
		"сравни", "сравнение", "разница", "отличие", "отличия",
		"в отличие", "по сравнению", "плюсы и минусы",
		"versus", " vs ", " vs", "vs ",
		"чем отличается",
		"и", " а также", "плюс ", "кроме того", "помимо",
	}
	for _, marker := range complexMarkers {
		if strings.Contains(q, marker) {
			return true
		}
	}

	if len(strings.Fields(q)) > 15 {
		return true
	}

	if strings.Count(q, "?") >= 2 {
		return true
	}

	return false
}

func DecomposeQuery(ctx context.Context, query string, cfg *config.Config) DecomposeResult { // разбивает сложный запрос на подвопросы
	fmt.Printf("[Decompose] Разбиваю сложный вопрос: '%s'\n", query)

	if !ShouldDecompose(query) {
		return DecomposeResult{
			Original:   query,
			SubQueries: []string{query},
			IsComplex:  false,
			Reason:     "запрос простой, декомпозиция не нужна",
		}
	}

	if cfg.LLM.Provider == "mock" || cfg.LLM.Provider == "" {
		result := DecomposeResult{
			Original:   query,
			SubQueries: fallbackDecompose(query),
			IsComplex:  true,
			Reason:     "использую fallback-декомпозицию (LLM не доступен)",
		}
		if result.IsComplex && len(result.SubQueries) > 1 {
			fmt.Printf("[Decompose] Получено %d подвопросов:\n", len(result.SubQueries))
			for i, sq := range result.SubQueries {
				fmt.Printf("   %d. '%s'\n", i+1, sq)
			}
		}
		return result
	}
	return decomposeWithLLM(ctx, query, cfg)
}

func decomposeWithLLM(ctx context.Context, query string, cfg *config.Config) DecomposeResult { // декомпозиция через ллм
	prompt := `Ты — помощник по декомпозиции запросов.
Разбей сложный вопрос на простые подвопросы для поиска в документации.

Правила:
1. Каждый подвопрос должен быть самодостаточным (можно искать отдельно)
2. Подвопросы должны покрывать все аспекты исходного вопроса
3. Максимум 4 подвопроса
4. Если вопрос уже простой — верни его как единственный подвопрос
5. Верни ТОЛЬКО JSON

Примеры:
Вопрос: "Сравни RAG и FileAuditor, в чем их разница и когда что использовать?"
Ответ: ["Что такое RAG?", "Что такое FileAuditor?", "Сравни RAG и FileAuditor", "Когда использовать RAG и когда FileAuditor?"]

Вопрос: "Что такое RAG?"
Ответ: ["Что такое RAG?"]

Вопрос: "Как установить и настроить DocSearch?"
Ответ: ["Как установить DocSearch?", "Как настроить DocSearch после установки?"]

Вопрос: %s

Ответь ТОЛЬКО JSON-массивом:`

	messages := []map[string]string{
		{"role": "system", "content": "Ты — помощник по декомпозиции запросов. Отвечай только JSON-массивом."},
		{"role": "user", "content": fmt.Sprintf(prompt, query)},
	}

	tempCfg := *cfg
	tempCfg.LLM.Temperature = 0.0
	tempCfg.LLM.MaxTokens = 200

	response, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, &tempCfg)
	if err != nil {
		return DecomposeResult{
			Original:   query,
			SubQueries: fallbackDecompose(query),
			IsComplex:  true,
			Reason:     "ошибка LLM, использую fallback: " + err.Error(),
		}
	}

	response = strings.TrimSpace(response) //  JSON
	jsonStr := extractJSONArray(response)
	if jsonStr == "" {
		return DecomposeResult{
			Original:   query,
			SubQueries: fallbackDecompose(query),
			IsComplex:  true,
			Reason:     "не удалось распарсить JSON, использую fallback",
		}
	}

	var subQueries []string
	if err := json.Unmarshal([]byte(jsonStr), &subQueries); err != nil {
		return DecomposeResult{
			Original:   query,
			SubQueries: fallbackDecompose(query),
			IsComplex:  true,
			Reason:     "ошибка парсинга JSON, использую fallback",
		}
	}

	filtered := []string{} // проверка что подвопросы не пустые
	for _, q := range subQueries {
		trimmed := strings.TrimSpace(q)
		if len(trimmed) > 3 {
			filtered = append(filtered, trimmed)
		}
	}

	if len(filtered) == 0 {
		return DecomposeResult{
			Original:   query,
			SubQueries: fallbackDecompose(query),
			IsComplex:  true,
			Reason:     "подвопросы пустые, использую fallback",
		}
	}

	result := DecomposeResult{
		Original:   query,
		SubQueries: filtered,
		IsComplex:  true,
		Reason:     "декомпозиция через LLM успешна",
	}

	if result.IsComplex && len(result.SubQueries) > 1 {
		fmt.Printf("[Decompose] Получено %d подвопросов:\n", len(result.SubQueries))
		for i, sq := range result.SubQueries {
			fmt.Printf("   %d. '%s'\n", i+1, sq)
		}
	}

	return result
}

func fallbackDecompose(query string) []string {
	q := strings.ToLower(query)

	if strings.Contains(q, "сравни") || strings.Contains(q, "сравнение") ||
		strings.Contains(q, "разница") || strings.Contains(q, "отличие") ||
		strings.Contains(q, "чем отличается") || strings.Contains(q, " vs ") {
		return []string{
			query + " — что такое первое понятие?",
			query + " — что такое второе понятие?",
			query + " — сравнение и различия",
			query + " — когда что использовать?",
		}
	}

	if strings.Contains(q, "как установить") || strings.Contains(q, "инструкция") ||
		strings.Contains(q, "пошагово") || strings.Contains(q, "настроить") {
		return []string{
			query + " — основные шаги",
			query + " — необходимые компоненты",
			query + " — проверка результата",
		}
	}

	if strings.Contains(q, "почему") || strings.Contains(q, "ошибка") ||
		strings.Contains(q, "не работает") {
		return []string{
			query + " — возможные причины",
			query + " — способы решения",
		}
	}

	if strings.Contains(q, " и ") && !strings.Contains(q, "сравни") { //если (и) то на части режу
		parts := strings.Split(q, " и ")
		if len(parts) >= 2 {
			subQueries := []string{}
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if len(part) > 3 {
					subQueries = append(subQueries, part+" — что это?")
				}
			}
			if len(subQueries) > 0 {
				return subQueries
			}
		}
	}

	return []string{query}
}

func extractJSONArray(s string) string { // извлекает JSON массив из строки
	start := strings.Index(s, "[")
	if start == -1 {
		return ""
	}

	bracketCount := 0
	for i := start; i < len(s); i++ {
		if s[i] == '[' {
			bracketCount++
		} else if s[i] == ']' {
			bracketCount--
			if bracketCount == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func IsSimpleQuery(query string) bool { // проверка на простой запрос
	q := strings.ToLower(strings.TrimSpace(query))

	if len(strings.Fields(q)) <= 3 { //короткий запрос
		return true
	}

	greetings := []string{"привет", "здравствуй", "добрый день", "спасибо"}
	for _, g := range greetings {
		if q == g || strings.HasPrefix(q, g+" ") {
			return true
		}
	}

	return false
}