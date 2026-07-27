package vector

import (
    "context"
    "testing"
)

func TestFakeVectorStoreSave(t *testing.T) {
    fake := NewFakeVectorStore()

    err := fake.Save(context.Background(), "documents", "test1", []float32{0.1, 0.2, 0.3}, map[string]interface{}{
        "chunk_text": "Тестовый текст",
        "doc_id": "test.md",
        "user_id": "user1",
        "page": 1,
    })
    if err != nil {
        t.Fatalf("Ошибка сохранения: %v", err)
    }

    if len(fake.Points) != 1 {
        t.Errorf("Ожидалось 1 точка, получено %d", len(fake.Points))
    }

    t.Log("FakeVectorStore.Save работает")
}

func TestFakeVectorStoreSearch(t *testing.T) {
    fake := NewFakeVectorStore()

    
    fake.Save(context.Background(), "documents", "test1", []float32{0.1, 0.2, 0.3}, map[string]interface{}{ //два чанка для разных людей
        "chunk_text": "Документ пользователя 1",
        "doc_id": "doc1.md",
        "user_id": "user1",
        "page": 1,
    })

    fake.Save(context.Background(), "documents", "test2", []float32{0.4, 0.5, 0.6}, map[string]interface{}{
        "chunk_text": "Документ пользователя 2",
        "doc_id": "doc2.md",
        "user_id":"user2",
        "page":1,
    })

    results, err := fake.Search(context.Background(), "documents", []float32{0.1, 0.2, 0.3}, 5, "user1")  // для user1
    if err != nil {
        t.Fatalf("Ошибка поиска: %v", err)
    }

    if len(results) != 1 {
        t.Errorf("Ожидался 1 результат для user1, получено %d", len(results))
    }

    payload, ok := results[0]["payload"].(map[string]interface{})
    if !ok {
        t.Error("Не удалось получить payload")
    } else {
        docID, ok := payload["doc_id"].(string)
        if !ok {
            t.Error("doc_id не строка")
        } else if docID != "doc1.md" {
            t.Errorf("Неверный документ: %s, ожидалось doc1.md", docID)
        }
    }

    t.Log("FakeVectorStore.Search фильтрует по user_id")
}

func TestFakeVectorStoreDelete(t *testing.T) {
    fake := NewFakeVectorStore()

    fake.Save(context.Background(), "documents", "test1", []float32{0.1, 0.2, 0.3}, map[string]interface{}{  //сохр тест чанк
        "chunk_text": "Тестовый документ",
        "doc_id": "test.md",
        "user_id": "user1",
        "page": 1,
    })

    if len(fake.Points) != 1 {
        t.Fatalf("Ожидалась 1 точка перед удалением, получено %d", len(fake.Points))
    }

    err := fake.Delete(context.Background(), "documents", nil)
    if err != nil {
        t.Fatalf("Ошибка удаления: %v", err)
    }

    if len(fake.Points) != 0 {
        t.Errorf("После удаления ожидалось 0 точек, получено %d", len(fake.Points))
    }

    t.Log("FakeVectorStore.Delete работает")
}

func TestFakeVectorStoreCreateCollection(t *testing.T) {
    fake := NewFakeVectorStore()

    err := fake.CreateCollection(context.Background(), "documents")
    if err != nil {
        t.Fatalf("Ошибка создания коллекции: %v", err)
    }

    t.Log("FakeVectorStore.CreateCollection работает")
}

func TestFakeVectorStorePing(t *testing.T) {
    fake := NewFakeVectorStore()

    err := fake.Ping(context.Background())
    if err != nil {
        t.Fatalf("Ошибка Ping: %v", err)
    }

    t.Log("FakeVectorStore.Ping работает")
}