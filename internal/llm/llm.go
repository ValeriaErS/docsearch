package llm

import (
	"bytes"
	"context"
	"docsearch/internal/config"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	llmTimeout = 100 * time.Second // ждем ответ 30 секунд
	maxRetries = 2
)

func init() {
	godotenv.Load() // ключик из env
}

func GetAnswerWithHistory(ctx context.Context, question string, chunks []string, docNames []string, pages []int, history []map[string]string, cfg *config.Config) (string, int, error) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return "", 0, fmt.Errorf("нет ключа")
	}

	url := cfg.LLM.BaseURL + "/chat/completions"

	context := "" // склеиваю чанки с указанием источника и страницы
	for i := 0; i < len(chunks); i++ {
		docName := "неизвестный документ"
		if i < len(docNames) && docNames[i] != "" {
			docName = docNames[i]
		}

		page := 1 // беру реальную страницу
		if i < len(pages) && pages[i] > 0 {
			page = pages[i]
		}
		context = context + fmt.Sprintf("\n--- Источник: %s, страница: %d ---\n%s", docName, page, chunks[i])
	}

	fmt.Printf("Контекст для LLM, страниц: %d\n", len(pages))
	if len(pages) > 0 {
		fmt.Printf("Страницы: %v\n", pages)
	}

	messages := []map[string]string{}

	systemPrompt := fmt.Sprintf(`Ты — помощник по документации.

Используй только сведения из предоставленного контекста.

Если информации недостаточно, ответь:
"В документации нет информации по этому вопросу."
ВАЖНО: ОТВЕЧАЙ ТОЛЬКО НА РУССКОМ ЯЗЫКЕ. НИКОГДА НЕ ИСПОЛЬЗУЙ АНГЛИЙСКИЙ.

Правила:

• Не придумывай факты.
• Не придумывай источники.
• Не придумывай номера страниц.
• Не используй знания вне контекста.
• Объединяй повторяющуюся информацию.
• Не повторяй одну мысль разными словами.
• Если ответ состоит из нескольких частей, располагай их в логическом порядке.
• После каждого смыслового абзаца обязательно указывай ссылку:
  [источник: имя_файла, страница N]
• Отвечай ТОЛЬКО на русском языке.
• Используй обычный текст без Markdown, таблиц и декоративных символов.

Структура ответа:

1. Краткий ответ.
2. Пояснение (если необходимо).
3. Источник после каждого абзаца.

Контекст из документов:
%s

Вопрос: %s
Ответ:`, context, question)

	messages = append(messages, map[string]string{
		"role":    "system",
		"content": "Ты помощник. Отвечай только по документам. Всегда указывай источник в формате [источник: название_файла.pdf, страница N].",
	})

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": systemPrompt,
	})

	start := 0 // добавляю историю
	if len(history) > 4 {
		start = len(history) - 4
	}
	for i := start; i < len(history); i++ {
		messages = append(messages, history[i])
	}

	data := map[string]interface{}{
		"model":       cfg.LLM.Model,
		"messages":    messages,
		"temperature": cfg.LLM.Temperature,
		"max_tokens":  cfg.LLM.MaxTokens,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-Title", "docsearch")

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Повторная попытка %d из %d\n", attempt+1, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		client := &http.Client{Timeout: llmTimeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("ошибка %d: %s", resp.StatusCode, string(body))
			continue
		}

		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				TotalTokens int `json:"total_tokens"`
			} `json:"usage"`
		}

		err = json.Unmarshal(body, &result)
		if err != nil {
			lastErr = err
			continue
		}

		if len(result.Choices) == 0 {
			lastErr = fmt.Errorf("нет ответа от модели")
			continue
		}

		answer := result.Choices[0].Message.Content

		answer = strings.ReplaceAll(answer, "**", "")
		answer = strings.ReplaceAll(answer, "*", "")
		answer = strings.ReplaceAll(answer, "###", "")
		answer = strings.ReplaceAll(answer, "##", "")
		answer = strings.ReplaceAll(answer, "#", "")

		answer = strings.ReplaceAll(answer, "[]", "")
		answer = strings.ReplaceAll(answer, "[ ]", "")
		answer = strings.ReplaceAll(answer, "()", "")

		re := regexp.MustCompile(`\n{3,}`)
		answer = re.ReplaceAllString(answer, "\n\n")

		answer = strings.TrimSpace(answer)

		tokensUsed := result.Usage.TotalTokens
		return answer, tokensUsed, nil
	}

	return "", 0, fmt.Errorf("не удалось получить ответ после %d попыток: %w", maxRetries, lastErr)
}

func GetAnswer(ctx context.Context, question string, chunks []string, cfg *config.Config) (string, int, error) {
	return GetAnswerWithHistory(ctx, question, chunks, []string{}, []int{}, []map[string]string{}, cfg)
}
