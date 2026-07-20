package llm

import (
	"fmt"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"regexp"
	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
}

func GetAnswerWithHistory(question string, chunks []string, docNames []string, history []map[string]string) (string, error) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("нет ключа")
	}

	url := "https://openrouter.ai/api/v1/chat/completions"

	
	context := ""
	for i := 0; i < len(chunks); i++ {
		docName := "документ"
		if i < len(docNames) && docNames[i] != "" && docNames[i] != "неизвестный документ" {
			docName = docNames[i]
		}
		context = context + fmt.Sprintf("\n--- %s ---\n%s", docName, chunks[i])
	}

	messages := []map[string]string{}

	systemPrompt := fmt.Sprintf(`Ты помощник. Отвечай на вопрос на русском языке, используя только информацию из документов ниже.
Если в документах нет ответа, скажи, что не знаешь.
Указывай источник сразу после каждого утверждения или абзаца в формате [название_файла].
Например: "FTPController предназначен для мониторинга документов [ftpcontroller_tech.pdf]".
Если название файла не указано в контексте — НЕ ставь источник и НЕ пиши "неизвестный документ" или "документ".
Не используй Markdown-разметку (звёздочки, решётки, таблицы).
Форматируй ответ: разделяй абзацы пустыми строками, используй переносы строк между пунктами.
Контекст из документов:
%s`, context)

	messages = append(messages, map[string]string{
		"role": "system",
		"content": systemPrompt,
	})

	
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	for i := start; i < len(history); i++ {
		messages = append(messages, history[i])
	}

	
	messages = append(messages, map[string]string{
		"role": "user",
		"content": question,
	})

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

	
	reTable := regexp.MustCompile(`(?m)^\|.*\|$`)
	answer = reTable.ReplaceAllString(answer, "")

	reSeparator := regexp.MustCompile(`(?m)\|[- :|]+\|`)
	answer = reSeparator.ReplaceAllString(answer, "")

	
	answer = strings.ReplaceAll(answer, "###", "")
    answer = strings.ReplaceAll(answer, "[неизвестный документ]", "")
	answer = strings.ReplaceAll(answer, "неизвестный документ", "")
	answer = strings.ReplaceAll(answer, "[неизвестный]", "")
    answer = strings.ReplaceAll(answer, "[документ]", "")  
    answer = strings.ReplaceAll(answer, "документ", "") 

	
	reBrackets := regexp.MustCompile(`\[\s*\]`)
	answer = reBrackets.ReplaceAllString(answer, "")

	
	for i := 1; i <= 10; i++ {
		old := fmt.Sprintf("%d.", i)
		new := fmt.Sprintf("\n\n%d.", i)
		answer = strings.ReplaceAll(answer, old, new)
	}

	answer = strings.ReplaceAll(answer, "Источники:", "\n\nИсточники:")

	
	re := regexp.MustCompile(`\n{3,}`)
	answer = re.ReplaceAllString(answer, "\n\n")

	
	answer = strings.TrimSpace(answer)



	return answer, nil
}


func GetAnswer(question string, chunks []string) (string, error) {
    return GetAnswerWithHistory(question, chunks, []string{}, []map[string]string{})
}