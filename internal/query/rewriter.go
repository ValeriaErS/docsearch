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

	
	systemPrompt := `Ты — помощник по переписыванию запросов.
Перепиши вопрос в четкий поисковый запрос.
Учти историю разговора.
Удали лишние слова (привет, спасибо).
Выдай ТОЛЬКО переписанный запрос.`

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

	rewritten = strings.TrimSpace(strings.Trim(rewritten, "\"'"))  // чищу ответ
	if len(rewritten) < 5 {
		return query, nil
	}

	return rewritten, nil
}

func (r *QueryRewriter) GenerateHyDE(ctx context.Context, query string) (string, error) {  // создает гипотетический ответ (метод HyDE)
	if r.cfg.LLM.Provider == "mock" {
		return query, nil
	}

	systemPrompt := `Ты — генератор гипотетических ответов.
Придумай примерный ответ на вопрос, КАК ЕСЛИ БЫ ОН БЫЛ В ДОКУМЕНТАЦИИ.
Не используй реальные факты — просто покажи, как выглядел бы ответ.
Используй технические термины.
Ответь на русском языке.`

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": query},
	}

	hypoAnswer, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, r.cfg)
	if err != nil {
		return query, nil
	}

	hypoAnswer = strings.TrimSpace(hypoAnswer)
	if len(hypoAnswer) < 20 {
		return query, nil
	}

	return hypoAnswer, nil
}