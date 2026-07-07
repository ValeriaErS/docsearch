package main
import (
    "fmt"
    "log"
    "docsearch/internal/vector"
)

func main() {
    client := vector.NewQdrantClient()    // создаю клиент
    err := client.Ping()
    if err != nil {
        log.Fatal("Qdrant не отвечает:", err)
    }
    fmt.Println("Qdrant работает!")

    err = client.CreateCollection("documents", 768)    // создаю коллекцию
    if err != nil {
        log.Fatal("Ошибка создания:", err)
    }
    fmt.Println("Коллекция создана")
} 