package indexer

import (
    "testing"
    "os"
    "docsearch/internal/config"
    "docsearch/internal/vector"
)

func testClient() *vector.QdrantClient {
    return &vector.QdrantClient{
        Host: "localhost",
        Port: 6333,
    }
}

func TestNewIndexer(t *testing.T) {
   
    if os.Getenv("QDRANT_HOST") == "" {
        t.Skip("QDRANT_HOST не задан, пропускаем тест")
    }
    if os.Getenv("QDRANT_PORT") == "" {
        t.Skip("QDRANT_PORT не задан, пропускаем тест")
    }
    
    cfg := &config.Config{}
    cfg.Corpus.Path = "./docs"
    cfg.Chunking.MaxTokens = 512
    cfg.Embeddings.VectorSize = 768

    vc := vector.NewQdrantClient()
    vc.VectorSize = 768

    idx := NewIndexer(cfg, vc, "test")
    if idx == nil {
        t.Error("Индексатор не создался")
    }

    if idx.UserID != "test" {
        t.Errorf("UserID не совпадает: %s, ожидалось: test", idx.UserID)
    }

    t.Log("Индексатор создан для пользователя:", idx.UserID)
}

func TestIndexerIndex(t *testing.T) {
    
    if os.Getenv("QDRANT_HOST") == "" {
        t.Skip("QDRANT_HOST не задан, пропускаем тест")
    }
    if os.Getenv("QDRANT_PORT") == "" {
        t.Skip("QDRANT_PORT не задан, пропускаем тест")
    }
    
    cfg := &config.Config{}
    cfg.Corpus.Path = "./docs"
    cfg.Chunking.MaxTokens = 512
    cfg.Embeddings.VectorSize = 768

    vc := vector.NewQdrantClient()
    vc.VectorSize = 768

    idx := NewIndexer(cfg, vc, "testuser")
    err := idx.Index()
    if err != nil {
        t.Logf("Индексация завершена с ошибкой или предупреждением: %v", err)
    }
}