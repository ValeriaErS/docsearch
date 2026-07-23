package rag

import (
    "testing"
    "os"
    "docsearch/internal/config"
    "docsearch/internal/vector"
)

func TestMock(t *testing.T) {

    if os.Getenv("QDRANT_HOST") == "" {
        t.Skip("QDRANT_HOST не задан, пропускаем тест")
    }
    if os.Getenv("QDRANT_PORT") == "" {
        t.Skip("QDRANT_PORT не задан, пропускаем тест")
    }
    
    client := vector.NewQdrantClient()
    if err := client.Ping(); err != nil {
        t.Skip("Qdrant не запущен, пропускаем тест")
    }
    
    cfg := config.Config{}
    cfg.LLM.Provider = "mock"
    cfg.Retrieval.TopK = 5
    cfg.Retrieval.MinScore = 0.2

    texts, docs, _, _, _, _ := Ask(cfg, "Что такое?", "test", []map[string]string{})

    if len(texts) == 0 {
        t.Log("Mock работает, но чанков не найдено")
    } else {
        t.Log("Mock работает, найдено чанков:", len(texts))
        t.Log("Документы:", docs)
    }
}

func TestMinScore(t *testing.T) {
    
    if os.Getenv("QDRANT_HOST") == "" {
        t.Skip("QDRANT_HOST не задан, пропускаем тест")
    }
    if os.Getenv("QDRANT_PORT") == "" {
        t.Skip("QDRANT_PORT не задан, пропускаем тест")
    }
    
    client := vector.NewQdrantClient()
    if err := client.Ping(); err != nil {
        t.Skip("Qdrant не запущен, пропускаем тест")
    }
    
    cfg := config.Config{}
    cfg.LLM.Provider = "mock"
    cfg.Retrieval.TopK = 5
    cfg.Retrieval.MinScore = 0.99 

    texts, _, _, answer, _, _ := Ask(cfg, "Что такое?", "test", []map[string]string{})

    if len(texts) == 0 {
        t.Log("min_score работает: чанков нет, ответ:", answer)
    } else {
        t.Log("Найдено чанков:", len(texts))
    }
}

func TestTenant(t *testing.T) {
  
    if os.Getenv("QDRANT_HOST") == "" {
        t.Skip("QDRANT_HOST не задан, пропускаем тест")
    }
    if os.Getenv("QDRANT_PORT") == "" {
        t.Skip("QDRANT_PORT не задан, пропускаем тест")
    }
    
    client := vector.NewQdrantClient()
    if err := client.Ping(); err != nil {
        t.Skip("Qdrant не запущен, пропускаем тест")
    }
    
    cfg := config.Config{}
    cfg.LLM.Provider = "mock"
    cfg.Retrieval.TopK = 5
    cfg.Retrieval.MinScore = 0.2

    _, docs, _, _, _, _ := Ask(cfg, "Что такое?", "Тест", []map[string]string{})

    t.Log("Поиск для пользователя Тест, найдено документов:", len(docs))
}