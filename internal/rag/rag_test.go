package rag

import (
    "testing"
    "docsearch/internal/vector"
    "context"
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