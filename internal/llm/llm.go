package llm

import (
    "fmt"
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "github.com/joho/godotenv"
)

func init() {  //ключик из env
    godotenv.Load()
}

func GetAnswer(question string, chunks []string) (string, error) {  //отправляет вопрос в llm
    apiKey := os.Getenv("LLM_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("нет ключа")
    }

    url := "https://openrouter.ai/api/v1/chat/completions"

    context := ""      // склеиваю все чанки в один
    for i := 0; i < len(chunks); i++ {
        context = context + fmt.Sprintf("\n[%d] %s", i+1, chunks[i])
    }

    prompt := fmt.Sprintf(`Ты помощник. Отвечай на вопрос, используя только информацию из документов ниже.
    Если в документах нет ответа, скажи, что не знаешь.
    В конце укажи источники в формате [1], [2] и т.д.
    Контекст из документов: %s
    Вопрос: %s
    Ответ:`, context, question)

    data := map[string]interface{}{    // тело запроса
        "model": "openrouter/free",
        "messages": []map[string]string{
            {"role": "user", "content": prompt},
        },
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

    req.Header.Set("Content-Type", "application/json")    // добавляю заголовки
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

    return result.Choices[0].Message.Content, nil
}