package llm

import (
    "fmt"
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "regexp"
    "github.com/joho/godotenv"
    "strings"
)

func init() {  //ключик из env
    godotenv.Load()
}
func GetAnswerWithHistory(question string,chunks []string, history []map[string]string) (string, error) {  //отправляет вопрос в llm
    apiKey := os.Getenv("LLM_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("нет ключа")
    } 
     


    url := "https://openrouter.ai/api/v1/chat/completions"

    context := ""      // склеиваю все чанки в один
    for i := 0; i < len(chunks); i++ {
        context = context + fmt.Sprintf("\n[%d] %s", i+1, chunks[i])
    }
    messages := []map[string]string{}

   messages = append(messages, map[string]string{
        "role": "system",
        "content": "Ты помощник. Отвечай по документам. Если не знаешь, скажи. Указывай источники [1], [2]. Контекст: " + context,
    })
    start:=0
    if len(history)>6{
        start=len(history)-6
    }
    for i:=start;i<len(history);i++{
        messages=append(messages,history[i])
    }
if len(history)==0 || history[len(history)-1]["content"]!=question{
    messages=append(messages,map[string]string{
        "role": "user",
        "content": question,
    })
}


    data := map[string]interface{}{    // тело запроса
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

    answer:=result.Choices[0].Message.Content
    answer=strings.ReplaceAll(answer, "**", "")
    answer = strings.ReplaceAll(answer, "*", "")
    
for i:=1;i<=10;i++{
    old:=fmt.Sprintf("%d.", i)
    new:=fmt.Sprintf ("\n%d.", i)
    answer=strings.ReplaceAll(answer,old,new)
}
    answer=strings.ReplaceAll(answer,"Источники:", "\n\nИсточники:")
    answer = strings.ReplaceAll(answer, "Таким образом,", "\n\nТаким образом,")
    answer = strings.ReplaceAll(answer, "Как это работает:", "\n\nКак это работает:")
    answer = strings.ReplaceAll(answer, "Назначение:", "\n\nНазначение:")
    answer = strings.ReplaceAll(answer, "Роль", "\nРоль")
    answer = strings.ReplaceAll(answer, "Обработка", "\nОбработка")
    answer = strings.ReplaceAll(answer, "Передача", "\nПередача")

    answer=strings.TrimSpace(answer)
    re := regexp.MustCompile(`\n{3,}`)
    answer = re.ReplaceAllString(answer, "\n\n")
    return answer,nil
}

func GetAnswer(question string, chunks []string) (string, error) {
    return GetAnswerWithHistory(question, chunks, []map[string]string{})
}
