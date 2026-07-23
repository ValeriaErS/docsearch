package rag

import (
    "testing"
    "docsearch/internal/config"
)
func TestMock(t *testing.T) {   // проверяю что mock работает
    
    cfg := config.Config{}
    cfg.LLM.Provider = "mock"
    cfg.Retrieval.TopK = 5
    cfg.Retrieval.MinScore = 0.2

    texts, docs, _, _, _, _ := Ask(cfg, "Что такое?", "test", []map[string]string{})

    if len(texts) == 0 {   // просто проверяю что что-то вернулось
        t.Log("Mock работает, но чанков не найдено")
    } else {
        t.Log("Mock работает, найдено чанков:", len(texts))
        t.Log("Документы:", docs)
    }
}

func TestMinScore(t *testing.T) {  // проверяю min_score (высокий порог)
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

func TestTenant(t *testing.T) {  // проверяю tenant 
    cfg := config.Config{}
    cfg.LLM.Provider = "mock"
    cfg.Retrieval.TopK = 5
    cfg.Retrieval.MinScore = 0.2

    _, docs, _, _, _, _ := Ask(cfg, "Что такое?", "Тест", []map[string]string{})

    t.Log("Поиск для пользователя Тест, найдено документов:", len(docs))
}