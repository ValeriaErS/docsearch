package main

import (
    "fmt"
    "docsearch/internal/embed"
    "docsearch/internal/vector"
)

func main() {
    client:= vector.NewQdrantClient()

    err := client.Ping()
    if err != nil {
        fmt.Println("бд не отвечает:", err)
        return
    }
    fmt.Println("бд работает")

 
    question := "Что такое RAG?"

    vec, err := embed.GetEmbedding(question)  // получаю эмбеддинг
    if err != nil {
        fmt.Println("Ошибка эмбеддинга:", err)
        return
    }

    vec32 := []float32{}    // перевожу для бд
    for i := 0; i < len(vec); i++ {
        vec32 = append(vec32, float32(vec[i]))
    }

    results, err := client.Search("documents", vec32, 5)   // топ 5 похожие чанки в бд
    if err != nil {
        fmt.Println("Ошибка поиска:", err)
        return
    }

    fmt.Println("\n Результаты поиска таковы:")
    fmt.Println("Вопрос:", question)
    fmt.Println("Найдено чанков:", len(results))

    for i, r := range results {
        fmt.Printf("\n--- Чанк %d (оценка: %.2f) ---\n", i+1, r["score"])
        payload := r["payload"].(map[string]interface{})
        text := payload["chunk_text"].(string)
        doc := payload["doc_id"].(string)

        preview := text             // покажу первые 200 символов
        if len(preview) > 200 {
            preview = preview[:200] + "..."
        }
        fmt.Println("Документ:", doc)
        fmt.Println("Текст:", preview)
    }
}