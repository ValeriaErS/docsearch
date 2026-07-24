package vector

import (
    "testing"
	"os"
    "context"
)

func testClient() *QdrantClient {  //клиент без env
    return &QdrantClient{
        Host: "localhost",
        Port: 6333,
    }
}

func TestNewQdrantClient(t *testing.T) {
    
    client := testClient()
    
    if client == nil {
        t.Error("Клиент не создался")
    }

    if client.Host == "" {
        t.Error("Хост не установлен")
    }

    if client.Port == 0 {
        t.Error("Порт не установлен")
    }

    t.Log("Клиент создан, хост:", client.Host, "порт:", client.Port)
}

func TestUrl(t *testing.T) {
    client := testClient()

    url := client.url("/test")
    expected := "http://localhost:6333/test"

    if url != expected {
        t.Errorf("URL не совпадает: %s, ожидалось: %s", url, expected)
    }

    t.Log("URL формируется правильно:", url)
}

func TestQdrantPing(t *testing.T) {
    
    if os.Getenv("QDRANT_HOST") == "" {
        t.Skip("QDRANT_HOST не задан, пропускаем тест")
    }
    if os.Getenv("QDRANT_PORT") == "" {
        t.Skip("QDRANT_PORT не задан, пропускаем тест")
    }
    
   client, err := NewQdrantClient()
   if err != nil {
    t.Skip("Ошибка подключения к qdrant:", err)
}
    if err := client.Ping(context.Background()); err != nil {
    t.Skip("qdrant не запущен, пропускаем тест")
}
    t.Log("qdrant доступен")
}