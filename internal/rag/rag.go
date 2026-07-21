package rag

import (
    "docsearch/internal/config"
    "docsearch/internal/embed"
    "docsearch/internal/llm"
    "docsearch/internal/vector"
)


func Ask(cfg config.Config, question string, userID string) ([]string, []string, []float64, string) {   // возвращает тексты, имена документов, оценки, ответ 
    
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