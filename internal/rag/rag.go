package rag

import (
    "fmt"
    "time"
    "docsearch/internal/config"
    "docsearch/internal/embed"
    "docsearch/internal/llm"
    "docsearch/internal/vector"
)

func Ask(cfg config.Config, question string, userID string) ([]string, []string, []float64, string, []int, map[string]float64) {
    startTotal := time.Now()

    fmt.Println("Провайдер LLM:", cfg.LLM.Provider)

    
    startEmbed := time.Now() //эмбеддинг
    vec, err := embed.GetEmbedding(question, &cfg)
    if err != nil {
        return []string{}, []string{}, []float64{}, "не могу понять ваш вопрос", []int{}, map[string]float64{}
    }
    embedDuration := time.Since(startEmbed).Seconds()

    
    vec32 := []float32{} //готовлю вектор
    for i := 0; i < len(vec); i++ {
        vec32 = append(vec32, float32(vec[i]))
    }

    
    client := vector.NewQdrantClient() //подключение к бд векторной
    client.VectorSize = cfg.Embeddings.VectorSize

    
    startSearch := time.Now()  //поиск
    results, err := client.Search("documents", vec32, cfg.Retrieval.TopK, userID)
    if err != nil || len(results) == 0 {
        return []string{}, []string{}, []float64{}, "ничего не нашла", []int{}, map[string]float64{}
    }
    searchDuration := time.Since(startSearch).Seconds()

   
    found := false //проверка порога
    for _, r := range results {
        if r["score"].(float64) >= cfg.Retrieval.MinScore {
            found = true
            break
        }
    }
    if !found {
        return []string{}, []string{}, []float64{}, "ничего не нашла (ниже порога)", []int{}, map[string]float64{}
    }
    filteredResults:=[]map[string]interface{}{}  //фильтрую чанки ниже порога
    for _,r:=range results{
        if r["score"].(float64)>=cfg.Retrieval.MinScore{
            filteredResults=append(filteredResults,r)
        }
    }
    if len(filteredResults)==0{
        return []string{},[]string{}, []float64{}, "В документации нет информации по этому вопросу", []int{}, map[string]float64{}
    }
    results=filteredResults

   
    texts := []string{}
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

    
    var answer string  //llm
    var llmDuration float64

    if cfg.LLM.Provider == "mock" {
        fmt.Println("Mock режим: реальный поиск выполнен, LLM возвращает тестовый ответ")
        answer = fmt.Sprintf(
            "Это тестовый ответ в режиме mock.\n\n"+
                "Вопрос: %s\n\n"+
                "Найдено %d релевантных чанков из документов: %v\n\n"+
                "В реальном режиме здесь был бы ответ от LLM.",
            question, len(texts), docs)
        llmDuration = 0
    } else {
        startLLM := time.Now()
        answer, err = llm.GetAnswerWithHistory(question, texts, docs, pages, []map[string]string{})
        if err != nil {
            return texts, docs, scores, "LLM не отвечает", pages, map[string]float64{}
        }
        llmDuration = time.Since(startLLM).Seconds()
    }

    
    totalDuration := time.Since(startTotal).Seconds()

    timings := map[string]float64{
        "total": totalDuration,
        "embed": embedDuration,
        "search": searchDuration,
        "llm": llmDuration,
    }

    return texts, docs, scores, answer, pages, timings
}