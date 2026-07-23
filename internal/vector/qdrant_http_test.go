package vector

import (
    "testing"
)

func TestNewQdrantClient(t *testing.T) {
    
    client := &QdrantClient{  //тестовый клиент
        Host: "localhost",
        Port: 6333,
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

    t.Log("Клиент создан, хост:", client.Host, "порт:", client.Port)
}

func TestUrl(t *testing.T) {
    client := &QdrantClient{
        Host: "localhost",
        Port: 6333,
    }

    url := client.url("/test")
    expected := "http://localhost:6333/test"

    if url != expected {
        t.Errorf("URL не совпадает: %s, ожидалось: %s", url, expected)
    }

    t.Log("URL формируется правильно:", url)
}