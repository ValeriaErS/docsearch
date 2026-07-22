package rag

import (
    "fmt"
    "time"
    "docsearch/internal/config"
    "docsearch/internal/embed"
    "docsearch/internal/llm"
    "docsearch/internal/vector"
)

func Ask(cfg config.Config, question string, userID string) ([]string, []string, []float64, string, []int, map[string]float64) {   // возвращает тексты, имена документов, оценки, ответ 
    startTotal := time.Now()
    
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
    mockPages := []int{1, 2, 3}
    mockAnswer := fmt.Sprintf(
        "Это тестовый ответ в режиме mock.\n\n"+
        "Вопрос: %s\n\n"+
        "Ответ: В демонстрационном режиме система работает без интернета. "+
        "Для получения реальных ответов установите в config.yml provider: openrouter и "+
        "запустите LM Studio для эмбеддингов.", question)
    
        timings := map[string]float64{
            "total": 0,
            "embed": 0,
            "search": 0,
            "llm": 0,
        }
    return  mockTexts, mockDocs, mockScores, mockAnswer, mockPages, timings
}
    startEmbed := time.Now()  //эмбединг
    vec, err := embed.GetEmbedding(question)
    if err != nil {
        return []string{}, []string{}, []float64{}, "не могу понять ваш вопрос", []int{}, map[string]float64{}
    }
    embedDuration := time.Since(startEmbed).Seconds()

   
    vec32 := []float32{} //вектор готовлю
    for i := 0; i < len(vec); i++ {
        vec32 = append(vec32, float32(vec[i]))
    }

   
    client := vector.NewQdrantClient()    // подключаюсь
    client.VectorSize = cfg.Embeddings.VectorSize

    
    startSearch := time.Now()  //поиск
    results, err := client.Search("documents", vec32, cfg.Retrieval.TopK, userID)
    if err != nil || len(results) == 0 {
        return []string{}, []string{}, []float64{}, "ничего не нашла", []int{}, map[string]float64{}
    }
    searchDuration := time.Since(startSearch).Seconds()
    
    
    found := false    // проверяю есть хотя бы один чанк с оценкой выше порога
    for _, r := range results {
        if r["score"].(float64) >= cfg.Retrieval.MinScore {
            found = true
            break
        }
    }
    if !found {
         return []string{}, []string{}, []float64{}, "ничего не нашла (ниже порога)", []int{}, map[string]float64{}
    }

    
    texts := []string{}   // достаю текст, имена файлов и оценки со стр
    docs := []string{}
    scores := []float64{}
    pages := []int{}

    for _, r := range results {
        payload := r["payload"].(map[string]interface{})
        texts = append(texts, payload["chunk_text"].(string))
        docs = append(docs, payload["doc_id"].(string))
        scores = append(scores, r["score"].(float64))
    

    page := 1
        if p, ok := payload["page"].(float64); ok && int(p) > 0 {
            page = int(p)
        }
        pages = append(pages, page)
    }
   
    startLLM := time.Now() //llm
    answer, err := llm.GetAnswerWithHistory(question, texts, docs, pages, []map[string]string{})
    if err != nil {
        return texts, docs, scores, "LLM не отвечает", pages, map[string]float64{}
    }
    llmDuration := time.Since(startLLM).Seconds()

    totalDuration := time.Since(startTotal).Seconds()

    timings := map[string]float64{
        "total": totalDuration,
        "embed": embedDuration,
        "search": searchDuration,
        "llm": llmDuration,
    }

    return texts, docs, scores, answer, pages, timings
}
