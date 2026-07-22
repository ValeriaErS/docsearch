package rag

import (
    "fmt"
    "docsearch/internal/config"
    "docsearch/internal/embed"
    "docsearch/internal/llm"
    "docsearch/internal/vector"
)

func Ask(cfg config.Config, question string, userID string) ([]string, []string, []float64, string) {   // возвращает тексты, имена документов, оценки, ответ 
    fmt.Println("Провайдер LLM:", cfg.LLM.Provider)
    
    if cfg.LLM.Provider == "mock" {
    fmt.Println("Mock режим активирован")

    mockTexts := []string{   
        "Это тестовый чанк 1 для демонстрации работы системы. Система работает в offline режиме.",
        "Это тестовый чанк 2. В реальном режиме здесь были бы документы из вашей папки docs/.",
        "Это тестовый чанк 3. Источники и страницы указаны для примера.",
    }
    mockDocs := []string{
        "mock_document1.pdf",
        "mock_document2.pdf",
        "mock_document3.pdf",
    }
    mockScores := []float64{0.99, 0.85, 0.70}
    mockAnswer := fmt.Sprintf(
        "Это тестовый ответ в режиме mock.\n\n"+
        "Вопрос: %s\n\n"+
        "Ответ: В демонстрационном режиме система работает без интернета. "+
        "Для получения реальных ответов установите в config.yml provider: openrouter и "+
        "запустите LM Studio для эмбеддингов.", question)
    
    return mockTexts, mockDocs, mockScores, mockAnswer
}
    vec, err := embed.GetEmbedding(question)  // делаю вектор из вопроса
    if err != nil {
        return []string{}, []string{}, []float64{}, "не могу понять ваш вопрос"
    }

   
    vec32 := []float32{}
    for i := 0; i < len(vec); i++ {
        vec32 = append(vec32, float32(vec[i]))
    }

   
    client := vector.NewQdrantClient()    // подключаюсь
    client.VectorSize = cfg.Embeddings.VectorSize

    
    results, err := client.Search("documents", vec32, cfg.Retrieval.TopK, userID)
    if err != nil || len(results) == 0 {
        return []string{}, []string{}, []float64{}, "ничего не нашла"
    }

    
    
    found := false    // проверяю есть хотя бы один чанк с оценкой выше порога
    for _, r := range results {
        if r["score"].(float64) >= cfg.Retrieval.MinScore {
            found = true
            break
        }
    }
    if !found {
        return []string{}, []string{}, []float64{}, "ничего не нашла (ниже порога)"
    }

    
    texts := []string{}   // достаю текст, имена файлов и оценки
    docs := []string{}
    scores := []float64{}

    for _, r := range results {
        payload := r["payload"].(map[string]interface{})
        texts = append(texts, payload["chunk_text"].(string))
        docs = append(docs, payload["doc_id"].(string))
        scores = append(scores, r["score"].(float64))
    }

   
    answer, err := llm.GetAnswer(question, texts)
    if err != nil {
        return texts, docs, scores, "LLM не отвечает"
    }

    return texts, docs, scores, answer
}