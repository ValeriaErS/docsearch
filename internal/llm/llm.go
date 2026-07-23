package llm

import (
    "time"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"github.com/joho/godotenv"
    "docsearch/internal/config"
    "context"
)
const (
    llmTimeout = 100 * time.Second  // ждем ответ 30 секунд
    maxRetries = 2               
)

func init() {
	godotenv.Load() // ключик из env
}

func GetAnswerWithHistory(ctx context.Context, question string, chunks []string, docNames []string, pages []int, history []map[string]string, cfg *config.Config) (string, error) {
    apiKey := os.Getenv("LLM_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("нет ключа")
    }

    url := "https://openrouter.ai/api/v1/chat/completions"

    
    context := ""          // склеиваю чанки с указанием источника и страницы
    for i := 0; i < len(chunks); i++ {
        docName := "неизвестный документ"
        if i < len(docNames) && docNames[i] != "" {
            docName = docNames[i]
        }
        
        page := 1   // беру реальную страницу
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

    systemPrompt := fmt.Sprintf(`Ты помощник. Отвечай строго на русском языке, используя только информацию из документов ниже.
Если в документах нет ответа, скажи: "В документации нет информации по этому вопросу".

ВАЖНО: После каждого утверждения или абзаца обязательно указывай источник в формате: [источник: название_файла.pdf, страница N]

Номер страницы бери из контекста (там написано "страница: X").
Не выдумывай страницы, которых нет в контексте.
Не выдумывай источники, которых нет в контексте.
Не используй звёздочки, решётки или таблицы.
Форматируй ответ: разделяй абзацы пустыми строками.

Контекст из документов:
%s

Вопрос: %s
Ответ:`, context, question)

    messages = append(messages, map[string]string{
        "role": "system",
        "content": "Ты помощник. Отвечай только по документам. Всегда указывай источник в формате [источник: название_файла.pdf, страница N].",
    })

    messages = append(messages, map[string]string{
        "role": "user",
        "content": systemPrompt,
    })

    
    start := 0        // добавляю историю
    if len(history) > 4 {
        start = len(history) - 4
    }
    for i := start; i < len(history); i++ {
        messages = append(messages, history[i])
    }

    data := map[string]interface{}{
        "model": cfg.LLM.Model,
        "messages": messages,
        "temperature":cfg.LLM.Temperature,
        "max_tokens": cfg.LLM.MaxTokens,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", err
    }

     req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
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

        return answer, nil
    }

    return "", fmt.Errorf("не удалось получить ответ после %d попыток: %w", maxRetries, lastErr)
}

func GetAnswer(ctx context.Context, question string, chunks []string, cfg *config.Config) (string, error) {
    return GetAnswerWithHistory(ctx, question, chunks, []string{}, []int{}, []map[string]string{}, cfg)
}