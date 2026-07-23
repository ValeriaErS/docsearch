package config

import (
    "testing"
)

func TestLoadConfig(t *testing.T) {

    cfg := &Config{}  // тестовый конфиг
    cfg.Corpus.Path = "./docs"
    cfg.Corpus.Formats = []string{"md", "txt"}
    cfg.Chunking.MaxTokens = 512
    cfg.Embeddings.VectorSize = 768
    cfg.Retrieval.TopK = 5
    cfg.Retrieval.MinScore = 0.2

    if cfg.Corpus.Path == "" {
        t.Error("Путь к документам не загружен")
    }

    if cfg.Chunking.MaxTokens == 0 {
        t.Error("Размер чанка не загружен")
    }

    if cfg.Embeddings.VectorSize == 0 {
        t.Error("Размер вектора не загружен")
    }

    if cfg.Retrieval.TopK == 0 {
        t.Error("TopK не загружен")
    }

    t.Log("Тестовый конфиг загружен")
}