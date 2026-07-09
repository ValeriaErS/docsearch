package main

import (
    "fmt"
    "docsearch/internal/chunk"
    "docsearch/internal/corpus"
    "docsearch/internal/embed"
    "docsearch/internal/vector"
    "github.com/google/uuid"
)

func main() {
    client:= vector.NewQdrantClient()

    err:= client.Ping()
    if err != nil {
        fmt.Println("бд не отвечает:", err)
        return
    }
    fmt.Println("бд работает")

    err = client.CreateCollection("documents", 768)
    if err != nil {
        fmt.Println("Ошибка создания коллекции:", err)
        return
    }
    fmt.Println("Коллекция создана")

    docs, err:= corpus.LoadDocuments("./docs")
    if err != nil {
        fmt.Println("Ошибка загрузки документов:", err)
        return
    }
    fmt.Println("Нашла документов:", len(docs))

    for _, doc := range docs {
        fmt.Println("\n---", doc.Name, "---")

        chunks := chunk.SplitIntelligent(doc.Text, doc.Name, 512)
        fmt.Println("Чанков:", len(chunks))

        for i, ch := range chunks {
            vec, err := embed.GetEmbedding(ch.Text)
            if err != nil {
                fmt.Println("Ошибка:", err)
                continue
            }

            vec32:= []float32{}
            for j:= 0; j < len(vec); j++ {
                vec32 = append(vec32, float32(vec[j]))
            }

            id:= uuid.New().String()   // создаю UUID 

            data:= map[string]interface{}{
                "doc_id":doc.Name,
                "chunk_text":ch.Text,
                "section":ch.Section,
                "level":ch.Level,
                "token_count":ch.TokenCount,
            }

            err = client.SavePoint("documents", id, vec32, data)
            if err != nil {
                fmt.Println("Ошибка сохранения:", err)
                continue
            }

            fmt.Printf("Чанк %d сохранён (%d токенов)\n", i, ch.TokenCount)
        }
    }

    fmt.Println("\n Индексация завершена!")
}