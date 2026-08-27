package query

import (
    "context"
    "docsearch/internal/config"
    "docsearch/internal/llm"
    "encoding/json"
    "regexp"
    "strings"
)

type Intent string //какие намерения пользователя смотрю

const (
    IntentDirect Intent = "direct" // прямой ответ без поиска
    IntentSearch Intent = "search" // нужен поиск по документам
)

type IntentResult struct { //результат классификации
    Intent   Intent `json:"intent"`
    Reason   string `json:"reason"`
    Query    string `json:"query"`    
    Original string `json:"original"` 
}

func ClassifyIntent(ctx context.Context, query string, cfg *config.Config) IntentResult { 
    if quickRules(query) {
        return IntentResult{
            Intent: IntentDirect,
            Reason: "приветствие или общее знание",
            Query:  query,
            Original: query,
        }
    }

    if cfg.LLM.Provider != "mock" {  // ллм классификация 
        return classifyWithLLM(ctx, query, cfg)
    }

    return IntentResult{
        Intent: IntentSearch,
        Reason: "по умолчанию ищем в документах",
        Query:  query,
        Original: query,
    }
}

func quickRules(query string) bool {
    q := strings.ToLower(strings.TrimSpace(query))

    greetings := []string{
        "привет", "здравствуй", "добрый день", "доброе утро",
        "добрый вечер", "hi", "hello", "hey", "здарова",
        "ку", "прив", "здравствуйте", "доброго времени суток",
    }
    for _, g := range greetings {
        if q == g || strings.HasPrefix(q, g+" ") {
            return true
        }
    }

    thanks := []string{"спасибо", "благодарю", "thanks", "thank you"}
    for _, t := range thanks {
        if q == t || strings.HasPrefix(q, t+" ") {
            return true
        }
    }

    if len(strings.Fields(q)) <= 3 && !strings.Contains(q, "?") {  // короткие сообщения без вопросительного знака
        return true
    }

    if regexp.MustCompile(`^[\W_]+$`).MatchString(q) {
        return true
    }

    return false
}

func classifyWithLLM(ctx context.Context, query string, cfg *config.Config) IntentResult {
    prompt := `Ты — классификатор намерений пользователя.
Определи, нужен ли поиск по документам для ответа на вопрос.

Правила:
- DIRECT — прямой ответ без поиска:
  * Приветствия, прощания, благодарности
  * Простые математические вычисления
  * Общие знания (не о компании/продукте)
  * Просьбы перевести/перефразировать

- SEARCH — нужен поиск по документам:
  * Вопросы о продукте, функциях, настройках
  * Вопросы о документации, API, инструкциях
  * Вопросы о компании, политиках, процессах

Верни JSON:
{"intent": "direct" или "search", "reason": "причина"}

Вопрос: ` + query

    messages := []map[string]string{
        {"role": "system", "content": "Ты классификатор. Отвечай только JSON."},
        {"role": "user", "content": prompt},
    }

    tempCfg := *cfg
    tempCfg.LLM.Temperature = 0.0
    tempCfg.LLM.MaxTokens = 60

    response, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, &tempCfg)
    if err != nil {
        return IntentResult{
            Intent: IntentSearch,
            Reason: "ошибка классификации, использую поиск",
            Query:  query,
            Original: query,
        }
    }

    response = strings.TrimSpace(response)  // JSON
    start := strings.Index(response, "{")
    end := strings.LastIndex(response, "}")
    if start == -1 || end == -1 || end <= start {
        return IntentResult{
            Intent: IntentSearch,
            Reason: "не удалось распарсить ответ",
            Query:  query,
            Original: query,
        }
    }

    jsonStr := response[start : end+1]
    var result struct {
        Intent string `json:"intent"`
        Reason string `json:"reason"`
    }

    if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
        return IntentResult{
            Intent: IntentSearch,
            Reason: "ошибка парсинга JSON",
            Query:  query,
            Original: query,
        }
    }

    intent := IntentSearch
    if strings.ToLower(result.Intent) == "direct" {
        intent = IntentDirect
    }

    return IntentResult{
        Intent:   intent,
        Reason:   result.Reason,
        Query:    query,
        Original: query,
    }
}

func (r IntentResult) IsDirect() bool {  //  проверяет надо ли отвечать без поиска
    return r.Intent == IntentDirect
}

func (r IntentResult) IsSearch() bool {  // проверяет нужен ли поиск
    return r.Intent == IntentSearch
}