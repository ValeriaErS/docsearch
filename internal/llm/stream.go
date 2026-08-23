package llm

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "time"

    "docsearch/internal/config"
)

type StreamChunk struct {
    Content string
    Done    bool
    Error   error
}

func GetAnswerStream(ctx context.Context, question string, chunks []string, docNames []string, pages []int, cfg *config.Config) (<-chan StreamChunk, error) {
    apiKey := os.Getenv("LLM_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("нет ключа API")
    }

    url := cfg.LLM.BaseURL + "/chat/completions"

    // Собираем контекст
    context := ""
    for i := 0; i < len(chunks); i++ {
        docName := "неизвестный документ"
        if i < len(docNames) && docNames[i] != "" {
            docName = docNames[i]
        }
        page := 1
        if i < len(pages) && pages[i] > 0 {
            page = pages[i]
        }
        context += fmt.Sprintf("\n--- Источник: %s, страница: %d ---\n%s", docName, page, chunks[i])
    }

    systemPrompt := fmt.Sprintf(`Ты — поисковый помощник. Отвечай ТОЛЬКО на русском языке.

Инструкция:
1. Прочитай контекст и найди ответ на вопрос.
2. Если ответ есть — напиши его КРАТКО (2-3 предложения).
3. Если ответа нет — напиши: "В документации нет информации по этому вопросу."
4. В конце каждого абзаца укажи источник: [источник: имя_файла, страница N]

НЕ ПИШИ НА АНГЛИЙСКОМ! ТОЛЬКО РУССКИЙ!

Контекст:
%s

Вопрос: %s

Твой ответ:`, context, question)

    messages := []map[string]string{
        {"role": "system", "content": "Ты помощник. Отвечай только по документам. Всегда указывай источник."},
        {"role": "user", "content": systemPrompt},
    }

    data := map[string]interface{}{
        "model":       cfg.LLM.Model,
        "messages":    messages,
        "temperature": cfg.LLM.Temperature,
        "max_tokens":  cfg.LLM.MaxTokens,
        "stream":      true,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)

    client := &http.Client{Timeout: 120 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        return nil, fmt.Errorf("ошибка %d: %s", resp.StatusCode, string(body))
    }

    ch := make(chan StreamChunk, 100)

    go func() {
        defer close(ch)
        defer resp.Body.Close()

        reader := bufio.NewReader(resp.Body)

        for {
            line, err := reader.ReadString('\n')
            if err != nil {
                if err == io.EOF {
                    ch <- StreamChunk{Done: true}
                    return
                }
                ch <- StreamChunk{Error: err, Done: true}
                return
            }

            line = strings.TrimSpace(line)
            if line == "" {
                continue
            }

            if !strings.HasPrefix(line, "data: ") {
                continue
            }

            data := strings.TrimPrefix(line, "data: ")
            if data == "[DONE]" {
                ch <- StreamChunk{Done: true}
                return
            }

            var response struct {
                Choices []struct {
                    Delta struct {
                        Content string `json:"content"`
                    } `json:"delta"`
                } `json:"choices"`
            }

            if err := json.Unmarshal([]byte(data), &response); err != nil {
                continue
            }

            if len(response.Choices) > 0 {
                content := response.Choices[0].Delta.Content
                if content != "" {
                    ch <- StreamChunk{Content: content}
                }
            }
        }
    }()

    return ch, nil
}