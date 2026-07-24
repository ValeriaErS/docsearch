package indexer

import (
    "testing"
    "docsearch/internal/config"
    "docsearch/internal/vector"
)

func TestIndexerWithFake(t *testing.T) {
    fakeClient := vector.NewFakeVectorStore()  //фейк клиент
    
    cfg := &config.Config{}
    cfg.Corpus.Path = "./docs"
    cfg.Chunking.MaxTokens = 512
    cfg.Embeddings.VectorSize = 768
    
    idx := NewIndexer(cfg, fakeClient, "testuser")
    
    if idx.VectorClient == nil {
        t.Error("Индексатор не получил клиент")
    }
    
    t.Log("Индексатор с фейковым qdrant создан")
}