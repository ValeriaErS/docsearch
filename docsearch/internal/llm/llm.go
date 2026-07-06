package llm

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

func GetAnswer(question string, chunks []string) (string, error) { //запрос к модели, даем ей вопрос и чанки
apiKey:= os.Getenv("LLM_API_KEY")
    if apiKey == "" {
    return "", fmt.Errorf("нет ключика")
    }

    url:= "https://openrouter.ai/api/v1/chat/completions"

    context:= ""
    for i:= 0; i < len(chunks); i++ { //собираю все чанки в контекст с номерами
        context = context + "\n[" + fmt.Sprint(i+1) + "] " + chunks[i]
    }

    prompt:= fmt.Sprintf(`Ты помощник. Отвечай на вопрос, используя информацию только из документов. Если не знаешь ответа, скажи, что не знаешь честно. В конце укажи источники в формате [1], [2].
    Контекст из документов:%s
	Вопрос: %s
	Ответ:`, context, question)

    data:= map[string]interface{}{
        "model": "openrouter/free",
        "messages": []map[string]string{
            {"role": "user", "content": prompt},
        },
        "temperature": 0.1,
        "max_tokens":   512,
    }

    jsonData, err:= json.Marshal(data) //структура в json
    if err!= nil {
        return "", err
    }

    req, err:= http.NewRequest("POST", url, bytes.NewBuffer(jsonData)) //http запрос
    if err!= nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json") //из конфига модели
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("HTTP-Referer", "http://localhost")
    req.Header.Set("X-Title", "docsearch")

    client:= &http.Client{} 
    resp, err:= client.Do(req)
    if err!= nil {
    return "", err
    }
    defer resp.Body.Close()

    body, err:= io.ReadAll(resp.Body)
    if err!= nil {
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

    err = json.Unmarshal(body, &result) // разбираю json в структуру
    if err != nil {
    return "", err
    }

    if len(result.Choices) == 0 {
    return "", fmt.Errorf("нет ответа от модели")
    }

    return result.Choices[0].Message.Content, nil
}