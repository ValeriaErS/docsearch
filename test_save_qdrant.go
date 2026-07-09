package main

import (
    "fmt"
    "docsearch/internal/vector"
)
func main() {
    client := vector.NewQdrantClient()

    err := client.Ping()   // проверяю, что бд доступна
    if err != nil {
        fmt.Println("бд не отвечает:", err)
        return
    }
    fmt.Println("бд работает")

    err = client.CreateCollection("documents", 768)   // создаю коллекцию для хранения векторов
    if err != nil {
        fmt.Println("Ошибка создания коллекции:", err)
        return
    }
    fmt.Println("Коллекция создана")

    vec:= make([]float32, 768)
    vec[0] = 0.1
    vec[1] = 0.2
    vec[2] = 0.3

    payload := map[string]interface{}{
        "doc_id": "test.md",
        "text": "Тестовый чанк",
    }

    id := "11111111-1111-1111-1111-111111111111"  // UUID

    err = client.SavePoint("documents", id, vec, payload)
    if err != nil {
        fmt.Println("Ошибка сохранения:", err)
        return
    }
    fmt.Println("Чанк сохранён")

    fmt.Println("Браузер:http://localhost:6333/dashboard#/collections/documents")
}