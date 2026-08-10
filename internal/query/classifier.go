package query

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

var jsonResponseRegex = regexp.MustCompile(`\{[^{}]*\}`)

func ClassifyQuery(ctx context.Context, query string, cfg *config.Config) (bool, error) {   //  определяет является ли запрос осмысленным для 
	systemPrompt := `Ты — классификатор запросов для RAG-системы поиска по документам.

Определи, является ли сообщение пользователя запросом, который имеет смысл передавать в систему поиска документов.

VALID, если пользователь:
- задаёт вопрос по теме документов;
- просит найти информацию;
- просит объяснить понятие;
- просит помочь разобраться с технической проблемой;
- формулирует запрос разговорно или с небольшими ошибками, но намерение понятно.

INVALID, если:
- это случайный набор символов;
- это бессмысленный текст;
- это чрезмерное повторение символов;
- это просто приветствие;
- это благодарность;
- это эмоциональное сообщение без конкретного запроса;
- это small talk, не связанный с поиском информации;
- невозможно определить, какую информацию пользователь хочет получить.

Примеры VALID:
"как работает RAG"
"что такое эмбеддинг"
"помоги разобраться с ошибкой"
"а что такое postgres"
"как подключить базу данных"
"объясни принцип работы векторного поиска"

Примеры INVALID:
"помогииииииииитеееееее"
"аааааааааааа"
"asdfghjkl"
"????????????"
"привет"
"спасибо"
"как дела?"
"бла бла бла"

Ответь ТОЛЬКО JSON:
{"valid":true}
или
{"valid":false}`

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": query},
	}
	tempCfg := *cfg  // маленькая модель для классификации
	tempCfg.LLM.Temperature = 0.0
	tempCfg.LLM.MaxTokens = 50

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, &tempCfg)
	if err != nil {
		return true, err 
	}

	response = strings.TrimSpace(response)
	match := jsonResponseRegex.FindString(response)

	if match == "" {
		return true, nil
	}

	var result struct {
		Valid bool `json:"valid"`
	}

	if err := json.Unmarshal([]byte(match), &result); err != nil {
		return true, nil
	}

	return result.Valid, nil
}