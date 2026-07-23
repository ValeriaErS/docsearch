package indexer

import (
    "testing"
    "docsearch/internal/config"
    "docsearch/internal/vector"
)

func TestNewIndexer(t *testing.T) {
    cfg := &config.Config{}
    cfg.Corpus.Path = "./docs"
    cfg.Chunking.MaxTokens = 512
    cfg.Embeddings.VectorSize = 768

    vc := &vector.QdrantClient{  //тестовый клиент
        Host: "localhost",
        Port: 6333,
        VectorSize: 768,
    }

    idx := NewIndexer(cfg, vc, "test")
    if idx == nil {
        t.Error("Индексатор не создался")
    }

    if idx.UserID != "test" {
        t.Errorf("UserID не совпадает: %s, ожидалось: test", idx.UserID)
    }

    t.Log("Индексатор создан для пользователя:", idx.UserID)
}