package rag

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/embed"
	"docsearch/internal/llm"
	"docsearch/internal/vector"
	"fmt"
	"time"
	"docsearch/internal/query"
	"docsearch/internal/retrieve"
	"docsearch/internal/rerank"
	"docsearch/internal/cache"
    "docsearch/internal/validate"
)

func Ask(ctx context.Context, cfg config.Config, question string, userID string, history []map[string]string, vectorClient vector.VectorStore) ([]string, []string, []float64, string, []int, []string, int, map[string]float64) {
	startTotal := time.Now()

	queryForSearch := question
	var multiQueryResults []map[string]interface{}
    rewriter := query.NewQueryRewriter(&cfg)

	fmt.Printf("Рерайтер создан, EnableRewriting=%v, EnableHyDE=%v\n", 
    cfg.Retrieval.EnableRewriting, cfg.Retrieval.EnableHyDE)

    if cfg.Retrieval.EnableHyDE {   // если включен HyDE то генерирую гипотетический ответ
        hypothetical, err := rewriter.GenerateHyDE(ctx, question)
        if err == nil && len(hypothetical) > 20 {
            queryForSearch = hypothetical
            fmt.Printf("HyDE сгенерирован (длина: %d символов)\n", len(hypothetical))
        } else if err != nil {
            fmt.Printf("HyDE ошибка: %v, используем оригинал\n", err)
        }
    }

    if cfg.Retrieval.EnableRewriting && queryForSearch == question {  //если HyDE не использовался
        rewritten, err := rewriter.Rewrite(ctx, question, history)
        if err == nil && len(rewritten) > 5 && rewritten != question {
            queryForSearch = rewritten
            fmt.Printf("Запрос переписан: '%s' → '%s'\n", question, rewritten)
        } else if err != nil {
            fmt.Printf("Rewriting ошибка: %v\n", err)
        }
    }

    if queryForSearch != question {  //финальный запрос
        fmt.Printf("Поиск по запросу: '%s'\n", queryForSearch)
    }

var fromCache bool   //проверка кэша поиска
var results []map[string]interface{}

if cfg.Cache.EnableSearchCache {
    searchCache := cache.GetSearchCache()
    if cached, ok := searchCache.Get(queryForSearch, userID); ok {
        results = cached
        fromCache = true
        fmt.Printf("Использую кэшированные результаты поиска (%d документов)\n", len(results))
    }
}

var embedDuration float64
var searchDuration float64

if fromCache {
    embedDuration = 0
    searchDuration = 0
    fmt.Printf("Использую кэшированные результаты (%d документов)\n", len(results))
} else {

    if cfg.Retrieval.EnableMultiQuery {  //muiti qery
        fmt.Printf("Запускаю Multi-Query поиск...\n")
        
        variants, err := query.GenerateMultiQueries(ctx, queryForSearch, &cfg)
        if err != nil {
            fmt.Printf("Ошибка генерации вариантов: %v\n", err)
            variants = []string{queryForSearch}
        }
        
        var allResults []map[string]interface{}
        for i, variant := range variants {
            fmt.Printf("Вариант %d: '%s'\n", i+1, variant)
            
            vec, err := embed.GetEmbedding(ctx, variant, &cfg)
            if err != nil {
                continue
            }
            
            vec32 := []float32{}
            for _, v := range vec {
                vec32 = append(vec32, float32(v))
            }
            
            res, err := vectorClient.Search(ctx, vector.CollectionName, vec32, cfg.Retrieval.TopK*2, userID)
            if err != nil || len(res) == 0 {
                continue
            }
            
            allResults = append(allResults, res...)
        }
        
        if len(allResults) > 0 {
            fusedResults := retrieve.ReciprocalRankFusion(allResults)
            if len(fusedResults) > cfg.Retrieval.TopK*3 {
                fusedResults = fusedResults[:cfg.Retrieval.TopK*3]
            }
            multiQueryResults = fusedResults  
            fmt.Printf("Multi-Query: объединено %d результатов\n", len(multiQueryResults))
        }
    }

    startEmbed := time.Now()   //эмбединг
    vec, err := embed.GetEmbedding(ctx, queryForSearch, &cfg)
    if err != nil {
        return []string{}, []string{}, []float64{}, "не могу понять ваш вопрос", []int{}, []string{}, 0, map[string]float64{}
    }
    embedDuration = time.Since(startEmbed).Seconds()

    vec32 := []float32{}
    for i := 0; i < len(vec); i++ {
        vec32 = append(vec32, float32(vec[i]))
    }

    if vectorClient == nil {
        var err error
        vectorClient, err = vector.NewQdrantClient()
        if err != nil {
            return []string{}, []string{}, []float64{}, "ошибка подключения к Qdrant", []int{}, []string{}, 0, map[string]float64{}
        }
        if qdrantClient, ok := vectorClient.(*vector.QdrantClient); ok {
            qdrantClient.VectorSize = cfg.Embeddings.VectorSize
        }
    }


    startSearch := time.Now()   //поиск векторный
    results, err = vectorClient.Search(ctx, vector.CollectionName, vec32, cfg.Retrieval.TopK, userID)
    if err != nil || len(results) == 0 {
        return []string{}, []string{}, []float64{}, "ничего не нашла", []int{}, []string{}, 0, map[string]float64{}
    }
    searchDuration = time.Since(startSearch).Seconds()

    if len(multiQueryResults) > 0 {
        results = multiQueryResults
        fmt.Printf("Использую Multi-Query результаты: %d документов\n", len(results))
    }
    
    textResults := []map[string]interface{}{}   // тестовый поиск HYBRID
    if cfg.Retrieval.HybridSearch {
        fmt.Printf("Запускаю полнотекстовый поиск по запросу: '%s'\n", queryForSearch)
        
        if qdrantClient, ok := vectorClient.(*vector.QdrantClient); ok {
            textResults, _ = qdrantClient.SearchText(ctx, vector.CollectionName, queryForSearch, cfg.Retrieval.TopK*3, userID)
        } else if fakeClient, ok := vectorClient.(*vector.FakeVectorStore); ok {
            textResults, _ = fakeClient.SearchText(ctx, vector.CollectionName, queryForSearch, cfg.Retrieval.TopK*3, userID)
        }
        
        if len(textResults) > 0 {
            fmt.Printf("Найдено %d результатов через полнотекстовый поиск\n", len(textResults))
        } else {
            fmt.Printf("Полнотекстовый поиск не вернул результатов\n")
        }
    }

    if len(textResults) > 0 && cfg.Retrieval.HybridSearch {
        fusedResults := retrieve.ReciprocalRankFusion(results, textResults)
        fmt.Printf("Объединено %d векторных + %d текстовых → %d результатов\n", 
            len(results), len(textResults), len(fusedResults))
        results = fusedResults
    } else {
        results = results[:cfg.Retrieval.TopK]
    }

    if cfg.Retrieval.EnableRerank && len(results) > cfg.Retrieval.TopK {      // реранкинг 
        fmt.Printf("Запускаю реранкинг: %d документов\n", len(results))

        documents := []string{}
        for _, r := range results {
            payload, ok := r["payload"].(map[string]interface{})
            if !ok {
                continue
            }
            chunkText, ok := payload["chunk_text"].(string)
            if ok && chunkText != "" {
                documents = append(documents, chunkText)
            }
        }

        if len(documents) > 0 {
            reranker := rerank.NewReranker()
            indices, _, err := reranker.Rerank(queryForSearch, documents, cfg.Retrieval.TopK)

            if err == nil && len(indices) > 0 {
                rerankedResults := []map[string]interface{}{}
                for _, idx := range indices {
                    if idx < len(results) {
                        rerankedResults = append(rerankedResults, results[idx])
                    }
                }
                if len(rerankedResults) > 0 {
                    results = rerankedResults
                    fmt.Printf("Реренкинг завершен: осталось %d документов\n", len(results))
                }
            } else {
                if len(results) > cfg.Retrieval.TopK {
                    results = results[:cfg.Retrieval.TopK]
                }
            }
        }
    }

    if cfg.Cache.EnableSearchCache && len(results) > 0 {
        searchCache := cache.GetSearchCache()
        searchCache.Set(queryForSearch, userID, results)
        fmt.Printf("Результаты поиска сохранены в кэш\n")
    }
}

found := false  //фильрация по порогу
for _, r := range results {
    score, ok := r["score"].(float64)
    if !ok {
        continue
    }
    if score >= cfg.Retrieval.MinScore {
        found = true
        break
    }
}
if !found {
    return []string{}, []string{}, []float64{}, "ничего не нашла (ниже порога)", []int{}, []string{}, 0, map[string]float64{}
}

filteredResults := []map[string]interface{}{}
for _, r := range results {
    score, ok := r["score"].(float64)
    if !ok {
        continue
    }
    if score >= cfg.Retrieval.MinScore {
        filteredResults = append(filteredResults, r)
    }
}

if len(filteredResults) == 0 {
    return []string{}, []string{}, []float64{}, "В документации нет информации по этому вопросу", []int{}, []string{}, 0, map[string]float64{}
}
results = filteredResults

texts := []string{}
docs := []string{}
scores := []float64{}
pages := []int{}
chunkIDs := []string{}

for _, r := range results {
    payload, ok := r["payload"].(map[string]interface{})
    if !ok {
        continue
    }
    chunkText, ok := payload["chunk_text"].(string)
    if !ok || chunkText == "" {
        continue
    }
    docID, ok := payload["doc_id"].(string)
    if !ok || docID == "" {
        continue
    }
    chunkScore, ok := r["score"].(float64)
    if !ok {
        continue
    }
    chunkID, ok := payload["chunk_id"].(string)
    if !ok {
        chunkID = "unknown"
    }
    page := 1
    if p, ok := payload["page"].(float64); ok && int(p) > 0 {
        page = int(p)
    }

    texts = append(texts, chunkText)
    docs = append(docs, docID)
    scores = append(scores, chunkScore)
    pages = append(pages, page)
    chunkIDs = append(chunkIDs, chunkID)
}
if len(texts) == 0 {
    return []string{}, []string{}, []float64{}, "В документации нет информации по этому вопросу", []int{}, []string{}, 0, map[string]float64{}
}

if cfg.Retrieval.EnableCompression && len(texts) > 0 {  
    fmt.Printf("Запускаю сжатие контекста...\n")
    
    compressedTexts, err := query.CompressChunks(ctx, texts, question, &cfg)
    if err == nil && len(compressedTexts) > 0 {
        texts = compressedTexts
        fmt.Printf("Контекст сжат: %d чанков\n", len(texts))
    } else {
        fmt.Printf("Сжатие не удалось: %v, использую оригиналы\n", err)
    }
}

var answer string  //llm
var llmDuration float64
tokensUsed := 0

if cfg.LLM.Provider == "mock" {
    answer = fmt.Sprintf(
        "Это тестовый ответ в режиме mock.\n\n"+
            "Вопрос: %s\n\n"+
            "Найдено %d релевантных чанков из документов: %v\n\n"+
            "В реальном режиме здесь был бы ответ от LLM.",
        question, len(texts), docs)
    llmDuration = 0
} else {
    fmt.Printf("Вызываю LLM с %d чанками\n", len(texts))
    startLLM := time.Now()
    var err error
    answer, tokensUsed, err = llm.GetAnswerWithHistory(ctx, question, texts, docs, pages, history, &cfg)
    if err != nil {
        fmt.Printf("Ошибка LLM: %v\n", err)
        return texts, docs, scores, "LLM не отвечает", pages, chunkIDs, 0, map[string]float64{}
    }
    fmt.Printf("LLM ответила, длина: %d символов\n", len(answer))
    llmDuration = time.Since(startLLM).Seconds()
    
    if cfg.Validation.EnableCitationValidator && len(answer) > 0 {
    fmt.Printf("Проверяю ссылки в ответе...\n")
    
    validatedAnswer, citations := validate.ValidateAnswer(answer, texts, docs)
    
    validCount := 0
    for _, c := range citations {
        if c.IsValid {
            validCount++
        }
    }
    
    if len(citations) > 0 {
        fmt.Printf(" Ссылок: %d, подтверждено: %d\n", len(citations), validCount)
    }
    
    if validatedAnswer != answer {
        answer = validatedAnswer
        fmt.Printf("Ответ очищен от неподтвержденных ссылок\n")
    }
}

}

totalDuration := time.Since(startTotal).Seconds()

timings := map[string]float64{
    "total":  totalDuration,
    "embed":  embedDuration,
    "search": searchDuration,
    "llm":    llmDuration,
}

return texts, docs, scores, answer, pages, chunkIDs, tokensUsed, timings
}
