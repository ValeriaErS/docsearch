package llm

import (
    "context"
    "strings"
    "time"
)

func IsLLMAvailable(ctx context.Context) bool {  // проверяет доступна ли ллм
    ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
    defer cancel()

    return true
}

func ShouldUseLLM(query string) bool { //  определяет стоит ли вызывать ллм для этого запроса
    q := strings.ToLower(strings.TrimSpace(query))

    if len(strings.Fields(q)) <= 2 { // для очень коротких запросов ллм не нужен
        return false
    }

    skipPhrases := []string{
        "привет", "здравствуй", "добрый день", "спасибо",
        "ок", "понял", "ясно", "хорошо", "hi", "hello",
    }
    for _, phrase := range skipPhrases {
        if q == phrase || strings.HasPrefix(q, phrase+" ") {
            return false
        }
    }

    return true
}