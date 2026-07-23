package config

import (
    "testing"
    "os"
)

func TestLoadConfig(t *testing.T) {
    
    if _, err := os.Stat("../../configs/config.yml"); os.IsNotExist(err) {
        t.Skip("config.yml не найден, пропускаем тест")
    }

    cfg, err := LoadConfig("../../configs/config.yml")
    if err != nil {
        t.Errorf("Ошибка загрузки конфига: %v", err)
        return
    }

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

    t.Log("Конфиг загружен из файла")
}