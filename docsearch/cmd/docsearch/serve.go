package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time" // ← добавила

    "docsearch/internal/config"
    "docsearch/internal/rag"
)

func runServe(configFile string, port string) {
    cfg, err := config.LoadConfig(configFile)
    if err != nil {
        fmt.Println("Ошибка загрузки настроек:", err)
        return
    }

    http.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(w, "Нужно использовать POST", http.StatusMethodNotAllowed)
            return
        }

        var body struct {
            Query string `json:"query"`
        }
        err := json.NewDecoder(r.Body).Decode(&body)
        if err != nil {
            http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
            return
        }

        if body.Query == "" {
            http.Error(w, "Нужно передать query", http.StatusBadRequest)
            return
        }

        startTime := time.Now() // ← засекаю время

        texts, docs, scores, answer := rag.Ask(*cfg, body.Query)

        found := false
        for i := 0; i < len(scores); i++ {
            if scores[i] >= cfg.Retrieval.MinScore {
                found = true
                break
            }
        }

        type Source struct {
            DocID   string  `json:"doc_id"`
            Score   float64 `json:"score"`
            Snippet string  `json:"snippet"`
        }

        var sources []Source
        if found {
            for i := 0; i < len(texts); i++ {
                snippet := texts[i]
                if len(snippet) > 100 {
                    snippet = snippet[:100] + "..."
                }
                sources = append(sources, Source{
                    DocID:   docs[i],
                    Score:   scores[i],
                    Snippet: snippet,
                })
            }
        }

        duration := time.Since(startTime).Milliseconds() // ← считаю время

        response := struct {
            Query      string   `json:"query"`
            Answer     string   `json:"answer"`
            Sources    []Source `json:"sources"`
            Model      string   `json:"model"`
            TokensUsed int      `json:"tokens_used"`
            DurationMs int64    `json:"duration_ms"`
        }{
            Query:      body.Query,
            Answer:     answer,
            Sources:    sources,
            Model:      cfg.LLM.Model,   // ← модель из конфига
            TokensUsed: 512,             // ← примерное значение
            DurationMs: duration,        // ← время
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    fmt.Println("Сервер запущен на порту", port)
    fmt.Println("Отправляй POST запрос на /ask с вопросом в поле query")

    err = http.ListenAndServe(port, nil)
    if err != nil {
        fmt.Println("Ошибка запуска сервера:", err)
    }
}