package indexer

import (
    "testing"
    "docsearch/internal/config"
    "docsearch/internal/vector"
    "docsearch/internal/corpus"
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
func TestHashText(t *testing.T) {
   
    doc := corpus.Document{
        Name: "test.md",
        Text: "Это тестовый текст для хеширования",
        Pages: nil,
    }

    cfg := &config.Config{}
    cfg.Chunking.MaxTokens = 512
    cfg.Chunking.OverlapTokens = 64
    cfg.Embeddings.Model = "test-model"
    cfg.Embeddings.VectorSize = 768
    cfg.LLM.Model = "test-llm"
    cfg.Retrieval.TopK = 5

    hash1 := hashText(doc, cfg)

    if hash1 == "" {
        t.Error("Хеш пустой")
    }

    if len(hash1) != 64 {
        t.Errorf("Неверная длина хеша: %d, ожидалось 64", len(hash1))
    }


    cfg.Chunking.MaxTokens = 1024   // меняю параметр и проверяю что хеш изменился
    hash2 := hashText(doc, cfg)

    if hash1 == hash2 {
        t.Error("Хеш не изменился при изменении параметров")
    }

    doc2 := corpus.Document{  //меняю текст
        Name: "test2.md",
        Text: "Другой текст для хеширования",
        Pages: nil,
    }
    hash3 := hashText(doc2, cfg)

    if hash2 == hash3 {
        t.Error("Хеш не изменился при изменении текста")
    }

    t.Log("Хеширование работает корректно")
}