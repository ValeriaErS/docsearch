package rag

import (
    "testing"
    "docsearch/internal/vector"
    "context"
    "docsearch/internal/config"  
)

func TestWithFakeQdrant(t *testing.T) { //проверяет с фейком
    fakeClient := vector.NewFakeVectorStore()
    
    fakeClient.Save(context.Background(), "documents", "test1", []float32{0.1, 0.2, 0.3}, map[string]interface{}{
    "chunk_text": "Это тестовый документ про RAG",
    "doc_id": "test.md",
    "page": 1,
    "user_id": "testuser",
})

fakeClient.Save(context.Background(), "documents", "test2", []float32{0.4, 0.5, 0.6}, map[string]interface{}{
    "chunk_text": "RAG - это Retrieval-Augmented Generation",
    "doc_id": "test.md",
    "page": 2,
    "user_id": "testuser",
})

results, err := fakeClient.Search(context.Background(), "documents", []float32{0.1, 0.2, 0.3}, 5, "testuser")
    if err != nil {
        t.Errorf("Ошибка поиска: %v", err)
    }
    
    if len(results) == 0 {
        t.Error("Фейк не вернул данные")
    }
    
    t.Log("Фейковый qdrant работает")
    t.Log("Найдено чанков:", len(results))
}

func TestContextCancel(t *testing.T) { //тест на отмену контекста
    client, err := vector.NewQdrantClient()
    if err != nil {
        t.Skip("qdrant не доступен")
        return
    }
    if err := client.Ping(context.Background()); err != nil {
        t.Skip("qdrant не запущен")
        return
    }

    cfg := config.Config{}
    cfg.LLM.Provider = "mock"
    cfg.Retrieval.TopK = 5
    cfg.Retrieval.MinScore = 0.2
    cfg.Embeddings.VectorSize = 768

    ctx, cancel := context.WithCancel(context.Background())
    cancel() 

   _, _, _, _, _, _, _ = Ask(ctx, cfg, "Что такое?", "test", []map[string]string{})
    t.Log("Тест на отмену контекста пройден")
}

func TestTenantIsolation(t *testing.T) { 
    fakeClient := vector.NewFakeVectorStore()   // фейковое хранилище

    err := fakeClient.Save(context.Background(), "documents", "doc1", []float32{0.1, 0.2, 0.3}, map[string]interface{}{   //документы для пользователя userA
        "chunk_text": "Это секретный документ пользователя А",
        "doc_id": "secret.doc",
        "user_id": "userA",
        "page": 1,
    })
    if err != nil {
        t.Fatalf("ошибка сохранения:%v", err)
    }

    results, err := fakeClient.Search(context.Background(), "documents", []float32{0.1, 0.2, 0.3}, 5, "userB") // userB не должен видеть документы userA
    if err != nil {
        t.Fatalf("ошибка поиска:%v", err)
    }

    if len(results) > 0 {
        t.Errorf("Ошибка: userB видит документы userA. Найдено: %d", len(results))
    } else {
        t.Log("userB ничего не видит")
    }

    results, err = fakeClient.Search(context.Background(), "documents", []float32{0.1, 0.2, 0.3}, 5, "userA")
    if err != nil {
        t.Fatalf("ошибка поиска:%v", err)
    }

    if len(results) == 0 {
        t.Error("Ошибка: userA не видит свои документы")
    } else {
        t.Logf("userA видит свои документы: %d", len(results))
    }
}
