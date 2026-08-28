package query

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"fmt"
	"strings"
)

type QueryRewriter struct { // улучшает запросы перед поиском, переспрашивает
	cfg *config.Config
}

func NewQueryRewriter(cfg *config.Config) *QueryRewriter { // создает нового переспрашивателя
	return &QueryRewriter{cfg: cfg}
}

func (r *QueryRewriter) Rewrite(ctx context.Context, query string, history []map[string]string) (string, error) {   // переписывает запрос с учетом истории диалога
	fmt.Printf("[Rewriter] Переписываю запрос: '%s'\n", query)

	if r.cfg.LLM.Provider == "mock" {
		return query, nil
	}

	if len(query) > 200 {
		return query, nil
	}

	// Сначала пробуем локальные правила - они работают всегда!
	rewritten := r.rewriteWithRules(query)
	if rewritten != query && len(rewritten) > 3 {
		fmt.Printf("[Rewriter] Локальное правило: '%s' → '%s'\n", query, rewritten)
		return rewritten, nil
	}

	// Если локальные правила не сработали, пробуем LLM
	systemPrompt := `Ты — помощник по переписыванию запросов для поиска.
Твоя задача — превратить вопрос пользователя в КОРОТКИЙ (2-5 слов) поисковый запрос на РУССКОМ языке.
Выдай ТОЛЬКО переписанный запрос, без кавычек, без лишних слов.

Примеры правильного переписывания:
Вопрос: "Как это работает?" → "Принцип работы FileAuditor"
Вопрос: "А как его настроить?" → "Настройка FileAuditor"
Вопрос: "Что такое FileAuditor?" → "FileAuditor описание"
Вопрос: "Где найти настройки?" → "Настройки FileAuditor"
Вопрос: "Сравни EndpointController и обычный контроллер" → "Сравнение EndpointController и контроллер"

Перепиши этот вопрос:`

	historyContext := ""
	count := 0
	for i := len(history) - 1; i >= 0 && count < 3; i-- {
		msg := history[i]
		historyContext = fmt.Sprintf("%s: %s\n", msg["role"], msg["content"]) + historyContext
		count++
	}

	userMessage := query
	if historyContext != "" {
		userMessage = fmt.Sprintf("История:\n%s\nВопрос: %s", historyContext, query)
	}

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userMessage},
	}

	rewritten, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, r.cfg)
	if err != nil {
		// Если LLM не сработал, возвращаем хотя бы очищенный запрос
		cleaned := r.cleanQuery(query)
		if cleaned != query {
			fmt.Printf("[Rewriter] LLM ошибка, очищенный запрос: '%s' → '%s'\n", query, cleaned)
			return cleaned, nil
		}
		return query, nil
	}

	rewritten = strings.TrimSpace(rewritten)
	rewritten = strings.Trim(rewritten, "\"'`")
	
	hasRussian := false
	for _, ch := range rewritten {
		if ch >= 0x0400 && ch <= 0x04FF { // русские буквы
			hasRussian = true
			break
		}
	}
	if !hasRussian && len(rewritten) > 0 {
		fmt.Printf("[Rewriter] Не русский ответ, используем локальное правило\n")
		cleaned := r.rewriteWithRules(query)
		if cleaned != query {
			fmt.Printf("[Rewriter] Локальное правило: '%s' → '%s'\n", query, cleaned)
			return cleaned, nil
		}
		return query, nil
	}

	forbidden := []string{"нет информации", "не найдено", "не знаю",
		"User Safety", "Response Safety",
		"I'm sorry", "I apologize", "I cannot"}
	for _, phrase := range forbidden {
		if strings.Contains(strings.ToLower(rewritten), strings.ToLower(phrase)) {
			fmt.Printf("[Rewriter] Запрещенная фраза, используем локальное правило\n")
			cleaned := r.rewriteWithRules(query)
			if cleaned != query {
				fmt.Printf("[Rewriter] Локальное правило: '%s' → '%s'\n", query, cleaned)
				return cleaned, nil
			}
			return query, nil
		}
	}
	
	if len(rewritten) > 50 {
		rewritten = rewritten[:50]
	}

	if len(rewritten) < 3 {
		cleaned := r.cleanQuery(query)
		if cleaned != query {
			fmt.Printf("[Rewriter] Короткий ответ, очищенный запрос: '%s' → '%s'\n", query, cleaned)
			return cleaned, nil
		}
		return query, nil
	}

	fmt.Printf("[Rewriter] Результат LLM: '%s' → '%s'\n", query, rewritten)
	return rewritten, nil
}

// rewriteWithRules - локальные правила перефразирования (работают всегда!)
func (r *QueryRewriter) rewriteWithRules(query string) string {
	q := strings.TrimSpace(query)
	lower := strings.ToLower(q)

	// ПРАВИЛО 1: "Сравни X и Y" → "Сравнение X и Y"
	if strings.HasPrefix(lower, "сравни") || strings.Contains(lower, "сравни") {
		rest := strings.TrimPrefix(strings.TrimPrefix(q, "Сравни"), "сравни")
		rest = strings.TrimSpace(rest)
		if len(rest) > 3 {
			return "Сравнение " + rest
		}
	}

	// ПРАВИЛО 2: "X это что?" → "Что такое X?"
	if strings.Contains(lower, "это что") {
		parts := strings.Split(q, "это что")
		if len(parts) >= 1 {
			subject := strings.TrimSpace(parts[0])
			if len(subject) > 2 {
				return "Что такое " + subject + "?"
			}
		}
	}

	// ПРАВИЛО 3: "X это?" → "Что такое X?"
	if strings.HasSuffix(lower, " это?") || strings.HasSuffix(lower, " это ?") {
		subject := strings.TrimSuffix(strings.TrimSuffix(q, " это?"), " это ?")
		subject = strings.TrimSpace(subject)
		if len(subject) > 2 {
			return "Что такое " + subject + "?"
		}
	}

	// ПРАВИЛО 4: "Как установить X?" → "Установка X"
	if strings.Contains(lower, "как установить") || strings.Contains(lower, "как поставить") {
		parts := strings.Split(q, "как установить")
		if len(parts) < 2 {
			parts = strings.Split(q, "как поставить")
		}
		if len(parts) >= 2 {
			subject := strings.TrimSpace(parts[1])
			subject = strings.TrimPrefix(subject, "этот")
			subject = strings.TrimPrefix(subject, "эту")
			subject = strings.TrimSpace(subject)
			if len(subject) > 2 {
				return "Установка " + subject
			}
		}
	}

	// ПРАВИЛО 5: "А как X?" → "X" (убираем "А как")
	if strings.HasPrefix(lower, "а как") || strings.HasPrefix(lower, "как") {
		cleaned := q
		cleaned = strings.TrimPrefix(cleaned, "А как ")
		cleaned = strings.TrimPrefix(cleaned, "Как ")
		cleaned = strings.TrimSpace(cleaned)
		if len(cleaned) > 3 && cleaned != q {
			return cleaned
		}
	}

	// ПРАВИЛО 6: для вопросов с "EndpointController" или "контроллер"
	if strings.Contains(lower, "endpointcontroller") || strings.Contains(lower, "контроллер") {
		// Определяем тип вопроса
		if strings.Contains(lower, "сравни") || strings.Contains(lower, "сравнение") {
			return "Сравнение EndpointController и контроллер"
		}
		if strings.Contains(lower, "установить") || strings.Contains(lower, "поставить") {
			return "Установка EndpointController"
		}
		if strings.Contains(lower, "что") || strings.Contains(lower, "это") {
			return "Что такое EndpointController?"
		}
		if strings.Contains(lower, "настроить") {
			return "Настройка EndpointController"
		}
		return "EndpointController информация"
	}

	// ПРАВИЛО 7: убираем слова-паразиты
	cleaned := q
	cleaned = strings.ReplaceAll(cleaned, "эту штуку", "")
	cleaned = strings.ReplaceAll(cleaned, "этот", "")
	cleaned = strings.ReplaceAll(cleaned, "эту", "")
	cleaned = strings.ReplaceAll(cleaned, "какую-то", "")
	cleaned = strings.ReplaceAll(cleaned, "какой-то", "")
	cleaned = strings.TrimSpace(cleaned)
	
	if len(cleaned) > 3 && cleaned != q {
		return cleaned
	}

	return query
}

// cleanQuery - базовое очищение запроса
func (r *QueryRewriter) cleanQuery(query string) string {
	cleaned := query
	cleaned = strings.ReplaceAll(cleaned, "А как ", "")
	cleaned = strings.ReplaceAll(cleaned, "Как ", "")
	cleaned = strings.ReplaceAll(cleaned, "эту штуку", "")
	cleaned = strings.ReplaceAll(cleaned, "этот", "")
	cleaned = strings.ReplaceAll(cleaned, "эту", "")
	cleaned = strings.ReplaceAll(cleaned, "какую-то", "")
	cleaned = strings.ReplaceAll(cleaned, "какой-то", "")
	cleaned = strings.TrimSpace(cleaned)
	
	if len(cleaned) < 3 {
		return query
	}
	return cleaned
}

func (r *QueryRewriter) GenerateHyDE(ctx context.Context, query string) (string, error) {  // создает гипотетический ответ (метод HyDE)
	fmt.Printf("[HyDE] Исходный вопрос: '%s'\n", query)

	if r.cfg.LLM.Provider == "mock" {
		return query, nil
	}

	if len(query) < 20 {
		fmt.Printf("[HyDE] Запрос слишком короткий, пропускаем\n")
		return query, nil
	}

	systemPrompt := `Ты — генератор поискового запроса.
Придумай КОРОТКИЙ (1 предложение) гипотетический ответ на вопрос на РУССКОМ языке.
Ответ должен быть информативным и содержать ключевые термины.

Примеры:
Вопрос: "Что такое FileAuditor?"
Ответ: "FileAuditor — это модуль для аудита файлов и контроля доступа"

Вопрос: "Как настроить доступ?"
Ответ: "Настройка доступа к файлам через FileAuditor"

Вопрос: "Какие функции у FileAuditor?"
Ответ: "FileAuditor выполняет поиск, аудит и маркировку файлов"

Вопрос: "Сравни EndpointController и обычный контроллер"
Ответ: "EndpointController специализируется на управлении API, а обычный контроллер управляет логикой приложения"

Вопрос пользователя:`

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": query},
	}

	hypoAnswer, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, r.cfg)
	if err != nil {
		fmt.Printf("[HyDE] Ошибка LLM: %v, используем оригинал\n", err)
		return query, nil
	}

	hypoAnswer = strings.TrimSpace(hypoAnswer)
	
	// Проверка на русский
	hasRussian := false
	for _, ch := range hypoAnswer {
		if ch >= 0x0400 && ch <= 0x04FF {
			hasRussian = true
			break
		}
	}
	if !hasRussian || len(hypoAnswer) < 15 {
		fmt.Printf("[HyDE] Ответ не подходит, используем оригинал\n")
		return query, nil
	}

	forbidden := []string{"нет информации", "не найдено", "не знаю", "no information"}
	for _, phrase := range forbidden {
		if strings.Contains(strings.ToLower(hypoAnswer), phrase) {
			fmt.Printf("[HyDE] Запрещенная фраза, используем оригинал\n")
			return query, nil
		}
	}

	fmt.Printf("[HyDE] Гипотетический ответ: '%s'\n", hypoAnswer)

	return hypoAnswer, nil
}