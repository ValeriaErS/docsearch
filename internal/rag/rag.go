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
	"docsearch/internal/monitor"
	"docsearch/internal/verify"
	"docsearch/internal/logger"
	"docsearch/internal/request"
	//"docsearch/internal/metrics"
	"docsearch/internal/alert"   
    "os" 
	"strings"
	
	
)
var redisCache *cache.RedisCache

func InitRedisCache(addr string, ttl time.Duration) {
    redisCache = cache.NewRedisCache(addr, "", 0, ttl)
	}
var errorAlertSent = false

func Ask(ctx context.Context, cfg config.Config, question string, userID string, history []map[string]string, vectorClient vector.VectorStore) ([]string, []string, []float64, string, []int, []string, int, map[string]float64) {
	//metrics.ActiveRequests.Inc()
   // defer metrics.ActiveRequests.Dec()
   fmt.Println("[rag.Ask] начало")
    fmt.Printf("Вопрос: %s\n", question)
    fmt.Printf("Пользователь: %s\n", userID)

	fmt.Println("=== трансформация запроса ===")
	fmt.Printf("1. исходный запрос пользователя: '%s'\n", question)

	startTotal := time.Now()

	requestID := request.GetRequestID(ctx) //логинг 
	pipelineLog := logger.NewPipelineLog(requestID, userID)
	validationStart := time.Now()

	validationService := query.NewValidationService(&cfg)  // валидация
	validation, err := validationService.Validate(ctx, question)

	if err != nil {
		pipelineLog.Validation = &logger.ValidationLog{
			Status:     "error",
			Reason:     err.Error(),
			DurationMs: time.Since(validationStart).Milliseconds(),
		}
		pipelineLog.Success = false
		pipelineLog.Error = err.Error()
		logger.GetPipelineLogger().Save(pipelineLog)
		return []string{}, []string{}, []float64{},
			"Не удалось проверить запрос. Попробуйте сформулировать вопрос иначе.",
			[]int{}, []string{}, 0, map[string]float64{}
	}

	pipelineLog.Validation = &logger.ValidationLog{
		Status:     string(validation.Status),
		Reason:     validation.Reason,
		DurationMs: time.Since(validationStart).Milliseconds(),
	}

	if validation.Status == query.StatusInvalid {
		pipelineLog.Success = false
		logger.GetPipelineLogger().Save(pipelineLog)
		return []string{}, []string{}, []float64{}, validation.Reason, []int{}, []string{}, 0, map[string]float64{}
	}
	intentResult := query.ClassifyIntent(ctx, question, &cfg)
	fmt.Printf("2. классификация намерения: %s (причина: %s)\n", intentResult.Intent, intentResult.Reason)
	
	if intentResult.IsDirect() {
    answer := "Привет! Я помогаю искать информацию в документации. Задайте вопрос, и я найду ответ среди загруженных документов."
    return []string{}, []string{}, []float64{}, answer, []int{}, []string{}, 0, map[string]float64{}
}

	complexity := query.ClassifyComplexity(question) // определяю сложность запроса и выбираю стратегию
	strategy := query.GetRetrievalStrategy(complexity)
	fmt.Printf("3. сложность запроса: %s\n", complexity)
	fmt.Printf("4. стратегия поиска: %s\n", strategy.Description)

	fmt.Printf("[%s] complexity=%s strategy=%q CandidateTopK=%d FinalTopK=%d rw=%v hyb=%v rr=%v mq=%v\n",
    requestID, complexity, strategy.Description,
    strategy.CandidateTopK, strategy.FinalTopK, 
    strategy.UseRewriting, strategy.UseHybrid, 
    strategy.UseRerank, strategy.UseMultiQuery)

	effectiveCfg := cfg
	effectiveCfg.Retrieval.CandidateTopK = strategy.CandidateTopK  
	effectiveCfg.Retrieval.RerankTopK = strategy.RerankTopK        
	effectiveCfg.Retrieval.FinalTopK = strategy.FinalTopK    
	effectiveCfg.Retrieval.EnableRewriting = strategy.UseRewriting
	effectiveCfg.Retrieval.HybridSearch = strategy.UseHybrid
	effectiveCfg.Retrieval.EnableRerank = strategy.UseRerank
	effectiveCfg.Retrieval.EnableMultiQuery = strategy.UseMultiQuery
	effectiveCfg.Retrieval.EnableHyDE = strategy.UseRewriting && cfg.Retrieval.EnableHyDE

	cfg = effectiveCfg

	monitorMetrics := &monitor.Metrics{}  //метрики
	monitorMetrics.StartNew(question, userID)
	monitorMetrics.SetModel(cfg.LLM.Model)
	monitorMetrics.SetQueryComplexity(string(complexity))
	monitorMetrics.SetRetrievalStrategy(strategy.Description)
	monitorMetrics.SetRetrievalRounds(1)

	queryForSearch := question
	fmt.Println("\n" + strings.Repeat("=", 60))
    fmt.Println("Запрос пользователя от а до я")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Printf("1. Исходный запрос пользователя:\n   \"%s\"\n", question)
   
    fmt.Println("\n2. Валидация и отчистка:")
    if len(question) < 3 {
        fmt.Println("Запрос слишком короткий")
    } else if len(question) > 2000 {
        fmt.Println("Запрос слишком длинный")
    } else {
        fmt.Println("Длина: допустимая")
        fmt.Println("Спецсимволы: нет")
        fmt.Println("Мусор: нет")
        fmt.Println("Результат: запрос чистый")
    }
  
    fmt.Println("\n3. INTENT CLASSIFICATION:")
    fmt.Printf("Намерение: %s\n", intentResult.Intent)
    fmt.Printf("Причина: %s\n", intentResult.Reason)
   
    if cfg.Retrieval.EnableRewriting {
        fmt.Println("\n4. QUERY REWRITING (перефразирование):")
        fmt.Printf("Было: \"%s\"\n", question)
        fmt.Printf("Стало: \"%s\"\n", queryForSearch)
    }
    
    if cfg.Retrieval.EnableHyDE && queryForSearch != question {
        fmt.Println("\n5. HYDE (Гипотетический ответ):")
        fmt.Printf("%s\n", queryForSearch)
    }
    
    fmt.Println("\n6. Финальный запрос QDRANT:")
    fmt.Printf(" → \"%s\"\n", queryForSearch)
    
  
    words := strings.Fields(queryForSearch)
    if len(words) > 0 {
        fmt.Println("\n7. Ключевые слова для поиска:")
        for i, word := range words {
            if i < 10 {
                fmt.Printf("   • %s\n", word)
            }
        }
        if len(words) > 10 {
            fmt.Printf("   ... и еще %d слов\n", len(words)-10)
        }
    }
    fmt.Println(strings.Repeat("=", 60))
	//var multiQueryResults []map[string]interface{}
	rewriter := query.NewQueryRewriter(&cfg)

	fmt.Printf("Рерайтер создан, EnableRewriting=%v, EnableHyDE=%v\n",
		cfg.Retrieval.EnableRewriting, cfg.Retrieval.EnableHyDE)

	if cfg.Retrieval.EnableHyDE { // если включен HyDE то генерирую гипотетический ответ
		hypothetical, err := rewriter.GenerateHyDE(ctx, question)
		if err == nil && len(hypothetical) > 20 {
			queryForSearch = hypothetical
			fmt.Printf("5. hyde генерация: '%s' → '%s'\n", question, hypothetical)
		} else if err != nil {
			fmt.Printf("HyDE ошибка: %v, используем оригинал\n", err)
		}
	}

	if cfg.Retrieval.EnableRewriting && queryForSearch == question { //если HyDE не использовался
		rewritten, err := rewriter.Rewrite(ctx, question, history)
		if err == nil && len(rewritten) > 5 && rewritten != question {
			queryForSearch = rewritten
			fmt.Printf("6. переписывание запроса: '%s' → '%s'\n", question, rewritten)
		} else if err != nil {
			fmt.Printf("Rewriting ошибка: %v\n", err)
		}
	}

	if queryForSearch != question { //финальный запрос
		fmt.Printf("9. финальный запрос в qdrant: '%s'\n", queryForSearch)
	}

	var fromCache bool  //проверка кэша поиска
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

	fmt.Println("=== начало поиска ===")

    startEmbed := time.Now() //получаю эмбеддинг
    vec, err := embed.GetEmbedding(ctx, queryForSearch, &cfg)
    if err != nil {
        return []string{}, []string{}, []float64{}, "не могу понять ваш вопрос", []int{}, []string{}, 0, map[string]float64{}
    }
    embedDuration = time.Since(startEmbed).Seconds()
    monitorMetrics.SetEmbeddingDuration(time.Since(startEmbed))

    vec32 := []float32{}
    for i := 0; i < len(vec); i++ {
        vec32 = append(vec32, float32(vec[i]))
    }

    if vectorClient == nil {   // проверка подключение к бд
        var err error
        vectorClient, err = vector.NewQdrantClient()
        if err != nil {
            return []string{}, []string{}, []float64{}, "ошибка подключения к Qdrant", []int{}, []string{}, 0, map[string]float64{}
        }
        if qdrantClient, ok := vectorClient.(*vector.QdrantClient); ok {
            qdrantClient.VectorSize = cfg.Embeddings.VectorSize
        }
    }
    
    candidateK := strategy.CandidateTopK
    if candidateK <= 0 {
        candidateK = 50 
    }
    fmt.Printf("Ищу %d кандидатов в Qdrant\n", candidateK)
    
    startSearch := time.Now()
    results, err = vectorClient.Search(ctx, vector.CollectionName, vec32, candidateK, userID)
    if err != nil || len(results) == 0 {
        return []string{}, []string{}, []float64{}, "ничего не нашла", []int{}, []string{}, 0, map[string]float64{}
    }
    searchDuration = time.Since(startSearch).Seconds()
    monitorMetrics.SetChunksFound(len(results))
    monitorMetrics.SetSearchDuration(time.Since(startSearch))

	if cfg.Retrieval.EnableDecomposition && strategy.UseDecomposition {
        if query.ShouldDecompose(queryForSearch) {
            fmt.Printf("8. декомпозиция запроса: '%s'\n", queryForSearch)
            
            decompResult := query.DecomposeQuery(ctx, queryForSearch, &cfg)
            
            if decompResult.IsComplex && len(decompResult.SubQueries) > 1 {
                fmt.Printf("   разбит на %d подвопросов:\n", len(decompResult.SubQueries))
                for i, sq := range decompResult.SubQueries {
                    fmt.Printf("   %d. '%s'\n", i+1, sq)
                }
                
                var allResults []map[string]interface{}
                
                for _, subQuery := range decompResult.SubQueries {
                    subVec, err := embed.GetEmbedding(ctx, subQuery, &cfg)
                    if err != nil {
                        fmt.Printf("Ошибка эмбеддинга для подзапроса '%s': %v\n", subQuery, err)
                        continue
                    }
                    
                    subVec32 := []float32{}
                    for _, v := range subVec {
                        subVec32 = append(subVec32, float32(v))
                    }
                    
                    subResults, err := vectorClient.Search(ctx, vector.CollectionName, subVec32, candidateK/2, userID)
                    if err != nil {
                        fmt.Printf("Ошибка поиска для подзапроса '%s': %v\n", subQuery, err)
                        continue
                    }
                    
                    allResults = append(allResults, subResults...)
                    fmt.Printf("Подзапрос '%s' → %d результатов\n", subQuery, len(subResults))
                }
                
                if len(allResults) > 0 {
                    results = retrieve.ReciprocalRankFusion(allResults)
                    if len(results) > candidateK {
                        results = results[:candidateK]
                    }
                    fmt.Printf("Декомпозиция: объединено %d результатов\n", len(results))
                } else {
                    fmt.Printf("Декомпозиция не дала результатов, использую оригинальный запрос\n")
                }
            }
        }
    }

    if cfg.Retrieval.EnableMultiQuery { // Multi-Query
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
        
            res, err := vectorClient.Search(ctx, vector.CollectionName, vec32, candidateK*2, userID)
            if err != nil || len(res) == 0 {
                continue
            }
            allResults = append(allResults, res...)
        }

        if len(allResults) > 0 {
            fusedResults := retrieve.ReciprocalRankFusion(allResults)
            if len(fusedResults) > candidateK {
                fusedResults = fusedResults[:candidateK]
            }
            results = fusedResults
            fmt.Printf("Multi-Query: объединено %d результатов\n", len(results))
        }
    }
	vectorResults := results

    textResults := []map[string]interface{}{} //текстовый и векторный поиск
    if cfg.Retrieval.HybridSearch {
    fmt.Printf("Текстовый поиск (BM25): ищу %d кандидатов\n", candidateK)
    
    if qdrantClient, ok := vectorClient.(*vector.QdrantClient); ok {
        textResults, err = qdrantClient.SearchText(ctx, vector.CollectionName, queryForSearch, candidateK, userID)
        if err != nil {
            fmt.Printf("Ошибка текстового поиска: %v\n", err)
        }
    } else if fakeClient, ok := vectorClient.(*vector.FakeVectorStore); ok {
        textResults, err = fakeClient.SearchText(ctx, vector.CollectionName, queryForSearch, candidateK, userID)
        if err != nil {
            fmt.Printf("Ошибка текстового поиска: %v\n", err)
        }
    }
    fmt.Printf("Текстовый поиск: найдено %d результатов\n", len(textResults))
}
  
    if len(textResults) > 0 && cfg.Retrieval.HybridSearch {
    fmt.Printf("Объединяю через Weighted RRF\n")
    
   vectorWeight := cfg.Retrieval.HybridWeights.Vector
textWeight := cfg.Retrieval.HybridWeights.Text
if vectorWeight <= 0 {
    vectorWeight = 1.0
}
if textWeight <= 0 {
    textWeight = 1.0
}
    
    fmt.Printf("Веса: Vector=%.1f, Text=%.1f\n", vectorWeight, textWeight)
    
    results = retrieve.WeightedReciprocalRankFusion(
        []float64{vectorWeight, textWeight},
        vectorResults,
        textResults,
    )
    
    if len(results) > candidateK {
        results = results[:candidateK]
    }
    fmt.Printf("После RRF: %d результатов\n", len(results))
}
    if cfg.Retrieval.EnableMMR && len(results) > 0 { //mmr
        fetchK := strategy.FinalTopK * 5
        if fetchK > len(results) {
            fetchK = len(results)
        }
        if fetchK > 50 {
            fetchK = 50 // огранич для скорости
        }

        mmrTopK := strategy.FinalTopK * 2
        if mmrTopK > len(results) {
            mmrTopK = len(results)
        }

        if mmrTopK > 0 && len(results) > mmrTopK {
            results = retrieve.MMRSelect(results, cfg.Retrieval.MMRLambda, mmrTopK, fetchK)
            fmt.Printf("MMR: оставлено %d разнообразных чанков (λ=%.2f)\n", len(results), cfg.Retrieval.MMRLambda)
        }
    }

rerankK := strategy.RerankTopK
if rerankK <= 0 {
    rerankK = 10
}

beforeRerank := len(results)
rerankStart := time.Now()


    if cfg.Retrieval.EnableRerank && len(results) > 1 {
    fmt.Printf("Запускаю реранкинг: %d → %d\n", len(results), rerankK)
    
    type docWithIndex struct {
        text  string
        index int
    }
    docsWithIdx := []docWithIndex{}
    
    for i, r := range results {
        payload, ok := r["payload"].(map[string]interface{})
        if !ok {
            continue
        }
        chunkText, ok := payload["chunk_text"].(string)
        if ok && chunkText != "" {
            docsWithIdx = append(docsWithIdx, docWithIndex{
                text:  chunkText,
                index: i,
            })
        }
    }

    if len(docsWithIdx) > 0 {
        documents := make([]string, len(docsWithIdx)) //извлекаю только тексты для реранкинга
        for i, d := range docsWithIdx {
            documents[i] = d.text
        }
        
        reranker := rerank.NewReranker(&cfg)
        indices, _, err := reranker.Rerank(ctx, queryForSearch, documents, rerankK)
        
        if err == nil && len(indices) > 0 {
            rerankedResults := []map[string]interface{}{}
            for _, idx := range indices {
                if idx < len(docsWithIdx) {
                  
                    originalIdx := docsWithIdx[idx].index //беру исходный рез по индексу
                    if originalIdx < len(results) {
                        rerankedResults = append(rerankedResults, results[originalIdx])
                    }
                }
            }
            if len(rerankedResults) > 0 {
                results = rerankedResults
                fmt.Printf("Реренкинг завершен: осталось %d документов\n", len(results))
            }
        }
    }
}
    pipelineLog.Rerank = &logger.RerankLog{  //логирую реранкинг
        BeforeCount: beforeRerank,
        AfterCount:  len(results),
        DurationMs:  time.Since(rerankStart).Milliseconds(),
    }

    finalK := strategy.FinalTopK
    if finalK <= 0 {
        finalK = 5 
    }
    
    if len(results) > finalK {
        results = results[:finalK]
        fmt.Printf("Финальный отбор: оставлено %d чанков для LLM\n", len(results))
    }
    if cfg.Cache.EnableSearchCache && len(results) > 0 {
        searchCache := cache.GetSearchCache()
        searchCache.Set(queryForSearch, userID, results)
        fmt.Printf("Результаты поиска сохранены в кэш\n")
    }
}


	if cfg.Retrieval.EnableRelevanceCheck && len(results) > 0 {
		fmt.Printf("Проверяю релевантность контекста...\n")

		checkTexts := []string{}  //  тексты для проверки
		for _, r := range results {
			payload, ok := r["payload"].(map[string]interface{})
			if !ok {
				continue
			}
			chunkText, ok := payload["chunk_text"].(string)
			if ok && chunkText != "" {
				checkTexts = append(checkTexts, chunkText)
			}
		}

		hasAnswer, confidence, err := retrieve.RelevanceCheck(ctx, question, checkTexts, &cfg)
		if err == nil && !hasAnswer {
			fmt.Printf("Контекст не содержит ответа (confidence: %.2f)\n", confidence)
			return []string{}, []string{}, []float64{},
				"В документации нет информации по этому вопросу.",
				[]int{}, []string{}, 0, map[string]float64{}
		}
		fmt.Printf("Контекст релевантен (confidence: %.2f)\n", confidence)
	}

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
			monitorMetrics.SetChunksAfterCompression(len(texts))
		} else {
			fmt.Printf("Сжатие не удалось: %v, использую оригиналы\n", err)
		}
	}
	
	var answer string   //llm
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
			monitorMetrics.SetError(err.Error())

			if !errorAlertSent {
        token := os.Getenv("TELEGRAM_BOT_TOKEN")
        chatID := os.Getenv("TELEGRAM_CHAT_ID")
        if token != "" && chatID != "" {
            bot := alert.NewTelegramBot(token, chatID)
            bot.Send("Ошибка LLM: " + err.Error())
            errorAlertSent = true
        }
    }

			return texts, docs, scores, "LLM не отвечает", pages, chunkIDs, 0, map[string]float64{}
		}
		fmt.Printf("[rag.Ask] Ответ от LLM получен\n")      // ← ЭТИ 3 СТРОЧКИ ВСТАВЬ
    fmt.Printf("Длина ответа: %d символов\n", len(answer))
    fmt.Printf("Токенов использовано: %d\n", tokensUsed)
		llmDuration = time.Since(startLLM).Seconds()
		monitorMetrics.SetLLMDuration(time.Since(startLLM))

		pipelineLog.LLM = &logger.LLMLog{ //логинг ллм
			Provider:   cfg.LLM.Provider,
			Model:      cfg.LLM.Model,
			ChunksSent: len(texts),
			TokensUsed: tokensUsed,
			DurationMs: time.Since(startLLM).Milliseconds(),
			Response:   answer,
		}
		

		if cfg.Verification.EnableAnswerVerification && len(answer) > 0 {
			fmt.Printf("Проверяю качество ответа...\n")

			result, err := verify.VerifyAnswer(ctx, question, answer, texts, &cfg)
			if err == nil && result != nil {
				if !result.IsAccurate {
					fmt.Printf("Ответ содержит ошибки: %s\n", result.Reason)
					if result.FixedAnswer != "" {
						answer = result.FixedAnswer
						fmt.Printf("Ответ исправлен\n")
					}
				} else {
					fmt.Printf("Ответ проверен (confidence: %.2f)\n", result.Confidence)
				}
			}
		}
		
		var citations []validate.Citation
		validCount := 0

		if cfg.Validation.EnableCitationValidator && len(answer) > 0 {
			fmt.Printf("Проверяю ссылки в ответе...\n")

			validatedAnswer, cit := validate.ValidateAnswer(answer, texts, docs)
			citations = cit

			validCount = 0
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
		
		report := validate.HallucinationReport{}

		if cfg.Validation.EnableHallucinationDetection && len(answer) > 0 {
			fmt.Printf("Проверяю ответ на галлюцинации...\n")

			report = validate.CheckHallucinations(answer, texts, docs)

			if report.HasHallucinations {
				fmt.Printf("Найдено %d неподтвержденных утверждений\n", report.Unverified)
				for _, claim := range report.UnverifiedList {
					fmt.Printf("'%s'\n", claim[:min(100, len(claim))])
				}

				if cfg.Validation.WarnOnHallucination {
					answer = answer + "\n\n Внимание: часть информации не подтверждена документами."
				}
			} else {
				fmt.Printf("Все утверждения подтверждены (%d проверено)\n", report.Verified)
			}
		}
		
		pipelineLog.PostProcessing = &logger.PostProcessingLog{
			CitationCount:  len(citations),
			ValidCitations: validCount,
			Hallucinations: report.Unverified,
			Verified:       true,
		}
		
	}
	
	totalDuration := time.Since(startTotal).Seconds()
	monitorMetrics.SetTokensUsed(tokensUsed)
	monitorMetrics.SetSources(docs)
	monitorMetrics.End(answer)

	monitor.GetCollector().Save(*monitorMetrics)
	
	pipelineLog.FinalAnswer = answer
	pipelineLog.Sources = docs
	pipelineLog.Success = true
	pipelineLog.DurationMs = time.Since(startTotal).Milliseconds()
	logger.GetPipelineLogger().Save(pipelineLog)
	

	timings := map[string]float64{
		"total":  totalDuration,
		"embed":  embedDuration,
		"search": searchDuration,
		"llm":    llmDuration,
	}

    fmt.Println("\n" + strings.Repeat("=", 60))
    fmt.Println("ОТВЕТ LLM")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Printf("%s\n", answer)
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println("")

	return texts, docs, scores, answer, pages, chunkIDs, tokensUsed, timings
}

func min(a, b int) int { // возвращает меньшее из двух чисел
	if a < b {
		return a
	}
	return b
}