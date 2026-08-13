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
	
	if r.cfg.LLM.Provider == "mock" {
		return query, nil
	}

	if len(query) > 200 {
		return query, nil
	}

	
	systemPrompt := `Ты — помощник по переписыванию запросов для поиска.
Твоя задача — превратить вопрос пользователя в КОРОТКИЙ (2-5 слов) поисковый запрос.

Примеры правильного переписывания:
Вопрос: "Как это работает?" → "Принцип работы FileAuditor"
Вопрос: "А как его настроить?" → "Настройка FileAuditor"
Вопрос: "Что такое FileAuditor?" → "FileAuditor описание"
Вопрос: "Где найти настройки?" → "Настройки FileAuditor"

Правила:
1. Запрос должен быть на русском языке
2. Используй ключевые термины из вопроса
3. Не добавляй слова "документация", "информация", "ответ"
4. Выдай ТОЛЬКО переписанный запрос, без кавычек

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

	rewritten, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, r.cfg)  	// прошу LLM переписать
	if err != nil {
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
    fmt.Printf("Query Rewriting вернул не русский ответ, используем оригинал\n")
    return query, nil
}

	forbidden := []string{"нет информации", "не найдено", "не знаю",
    "User Safety", "Response Safety",
    "I'm sorry", "I apologize", "I cannot",}
	for _, phrase := range forbidden {
		if strings.Contains(strings.ToLower(rewritten), strings.ToLower(phrase)) {
			fmt.Printf("Рерайтер вернул плохой ответ: '%s', используем оригинал\n", rewritten)
			return query, nil
		}
	}
	if len(rewritten) > 50 { 
    rewritten = rewritten[:50]
}

	if len(rewritten) < 3 {
		return query, nil
	}

	return rewritten, nil
}

func (r *QueryRewriter) GenerateHyDE(ctx context.Context, query string) (string, error) {  // создает гипотетический ответ (метод HyDE)
	if r.cfg.LLM.Provider == "mock" {
		return query, nil
	}

	systemPrompt := `Ты — генератор поискового запроса.
Придумай КОРОТКИЙ (1 предложение) гипотетический ответ на вопрос.

Примеры:
Вопрос: "Что такое FileAuditor?"
Ответ: "FileAuditor — это модуль для аудита файлов и контроля доступа"

Вопрос: "Как настроить доступ?"
Ответ: "Настройка доступа к файлам через FileAuditor"

Вопрос: "Какие функции у FileAuditor?"
Ответ: "FileAuditor выполняет поиск, аудит и маркировку файлов"

Правила:
1. Ответ должен быть на русском языке
2. Не используй слова "документация", "информация"
3. Выдай ТОЛЬКО гипотетический ответ, без кавычек

Вопрос пользователя:`

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": query},
	}

	hypoAnswer, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, r.cfg)
	if err != nil {
		return query, nil
	}

	
	forbidden := []string{"нет информации", "не найдено", "не знаю", "no information"}
	for _, phrase := range forbidden {
		if strings.Contains(strings.ToLower(hypoAnswer), phrase) {
			fmt.Printf("HyDE вернул плохой ответ, используем оригинал\n")
			return query, nil
		}
	}

	if len(hypoAnswer) < 10 {
		return query, nil
	}

	return hypoAnswer, nil
}