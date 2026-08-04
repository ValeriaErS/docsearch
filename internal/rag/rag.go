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
)

func Ask(ctx context.Context, cfg config.Config, question string, userID string, history []map[string]string, vectorClient vector.VectorStore) ([]string, []string, []float64, string, []int, []string, int, map[string]float64) {
	startTotal := time.Now()

	queryForSearch := question
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

	startEmbed := time.Now() //эмбеддинг
	vec, err := embed.GetEmbedding(ctx, queryForSearch, &cfg)
	if err != nil {
		return []string{}, []string{}, []float64{}, "не могу понять ваш вопрос", []int{}, []string{}, 0, map[string]float64{}
	}
	embedDuration := time.Since(startEmbed).Seconds()

	vec32 := []float32{} //готовлю вектор
	for i := 0; i < len(vec); i++ {
		vec32 = append(vec32, float32(vec[i]))
	}

	if vectorClient == nil { // если клиент не передан создаю реальный бд
		var err error
		vectorClient, err = vector.NewQdrantClient()
		if err != nil {
			return []string{}, []string{}, []float64{}, "ошибка подключения к Qdrant", []int{}, []string{}, 0, map[string]float64{}
		}
		if qdrantClient, ok := vectorClient.(*vector.QdrantClient); ok {
			qdrantClient.VectorSize = cfg.Embeddings.VectorSize
		}
	}

	startSearch := time.Now() //поиск
	results, err := vectorClient.Search(ctx, vector.CollectionName, vec32, cfg.Retrieval.TopK, userID)
	if err != nil || len(results) == 0 {
		return []string{}, []string{}, []float64{}, "ничего не нашла", []int{}, []string{}, 0, map[string]float64{}
	}
	searchDuration := time.Since(startSearch).Seconds()
	
	textResults := []map[string]interface{}{}  //текстовый поиск
	if cfg.Retrieval.HybridSearch {
	fmt.Printf("Запускаю полнотекстовый поиск по запросу: '%s'\n", queryForSearch)
	
	if qdrantClient, ok := vectorClient.(*vector.QdrantClient); ok {  // пробую получить текстовые результаты
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


var fusedResults []map[string]interface{}  //объединение
if len(textResults) > 0 && cfg.Retrieval.HybridSearch {
	fusedResults = retrieve.ReciprocalRankFusion(results, textResults)
	fmt.Printf("Объединено %d векторных + %d текстовых → %d результатов\n", 
		len(results), len(textResults), len(fusedResults))
	results = fusedResults
} else {
	results = results[:cfg.Retrieval.TopK] 
}

	found := false              //проверка порога
	for _, r := range results { //  приведения типов все v, ok := проверены
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

	filteredResults := []map[string]interface{}{} //фильтрую чанки ниже порога
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

	var answer string //llm
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
