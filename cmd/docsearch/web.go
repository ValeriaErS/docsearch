package main

import (
	"fmt"
    "encoding/json"
    "net/http"
    "docsearch/internal/embed"
    "docsearch/internal/llm"
    "docsearch/internal/vector"
    "docsearch/internal/config"
)

func runWeb(cfg *config.Config, port string) {     //запуск сервера
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "web/index.html")
    })

    http.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {  // обрабатываю вопросы которые приходят из чата
        if r.Method != "POST" {
            http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
            return
        }

        var req struct {               // читаю вопрос из тела
            Query string `json:"query"`
        }
        json.NewDecoder(r.Body).Decode(&req)

        if req.Query == "" {
            http.Error(w, "Пустой вопрос", http.StatusBadRequest)
            return
        }

       answer, sources := findAnswer(req.Query)

        w.Header().Set("Content-Type", "application/json")     // отправляю ответ
        json.NewEncoder(w).Encode(map[string]interface{}{
            "answer":  answer,
            "sources": sources,
        })
    })

    fmt.Println("Сайт запущен: http://localhost" + port)
    http.ListenAndServe(port, nil)
}

func findAnswer(question string) (string, []map[string]interface{}) {   //ищет ответ на вопрос в документации
    client := vector.NewQdrantClient()
    client.VectorSize = 999
   

    vec, err := embed.GetEmbedding(question)
    if err != nil {
        return "Ошибка: не могу понять вопрос", nil
    }

    vec32 := []float32{}
    for _, v := range vec {
        vec32 = append(vec32, float32(v))
    }

    results, err := client.Search("documents", vec32, 10)   // ищу похожие чанки
    if err != nil || len(results) == 0 {
        return "Ничего не нашла", nil
    }

    context := []string{}
    sources := []map[string]interface{}{}

    for _, r := range results {
        payload := r["payload"].(map[string]interface{})
        text := payload["chunk_text"].(string)
        context = append(context, text)
        sources = append(sources, map[string]interface{}{
            "doc_id": payload["doc_id"],
            "score":  r["score"],
        })
    }

seen := map[string]bool{}
uniqueSources := []map[string]interface{}{}
for _, s := range sources {
    docID := s["doc_id"].(string)
    if !seen[docID] {
        seen[docID] = true
        uniqueSources = append(uniqueSources, s)
    }
}
sources = uniqueSources
    answer, err := llm.GetAnswer(question, context)    // отправляю в llm
    if err != nil {
        return "Ошибка: нейросеть не отвечает", sources
    }

    return answer, sources
}