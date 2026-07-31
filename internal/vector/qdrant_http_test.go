package vector

import (
	"context"
	"os"
	"testing"
)

func testClient() *QdrantClient {  //клиент без env
	return &QdrantClient{
		Host: "localhost",
		Port: 6333,
	}
}

func TestNewQdrantClient(t *testing.T) {
	client, err := NewQdrantClient()
	if err != nil {
		t.Skip("Qdrant не доступен, пропускаем тест")
	}

	if client == nil {
		t.Error("Клиент не создался")
	}
	if client.Host == "" {
		t.Error("Хост не установлен")
	}
	if client.Port == 0 {
		t.Error("Порт не установлен")
	}
	t.Log("Клиент создан")
}

func TestUrl(t *testing.T) {
	client := testClient()
	url := client.url("/test")
	expected := "http://localhost:6333/test"
	if url != expected {
		t.Errorf("URL не совпадает: %s, ожидалось: %s", url, expected)
	}
	t.Log("URL формируется правильно")
}

func TestQdrantPing(t *testing.T) {
	if os.Getenv("QDRANT_HOST") == "" {
		t.Skip("QDRANT_HOST не задан")
	}
	if os.Getenv("QDRANT_PORT") == "" {
		t.Skip("QDRANT_PORT не задан")
	}
	client, err := NewQdrantClient()
	if err != nil {
		t.Skip("Ошибка подключения")
	}
	if err := client.Ping(context.Background()); err != nil {
		t.Skip("Qdrant не запущен")
	}
	t.Log("Qdrant доступен")
}