package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load() // ключик из env
}

func GetAnswerWithHistory(question string, chunks []string, docNames []string, pages []int, history []map[string]string) (string, error) {
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
        "model": "openrouter/free",
        "messages": messages,
        "temperature": 0.1,
        "max_tokens": 1024,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("HTTP-Referer", "http://localhost")
    req.Header.Set("X-Title", "docsearch")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    if resp.StatusCode != 200 {
        return "", fmt.Errorf("ошибка %d: %s", resp.StatusCode, string(body))
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
        return "", err
    }

    if len(result.Choices) == 0 {
        return "", fmt.Errorf("нет ответа от модели")
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

func GetAnswer(question string, chunks []string) (string, error) {
	return GetAnswerWithHistory(question, chunks, []string{}, []int{}, []map[string]string{})
}