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
	prommetrics "docsearch/internal/metrics"
)

func Ask(ctx context.Context, cfg config.Config, question string, userID string, history []map[string]string, vectorClient vector.VectorStore) ([]string, []string, []float64, string, []int, []string, int, map[string]float64) {
	prommetrics.ActiveRequests.Inc()
    defer prommetrics.ActiveRequests.Dec()

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

	complexity := query.ClassifyComplexity(question) // определяю сложность запроса и выбираю стратегию
	strategy := query.GetRetrievalStrategy(complexity)

	fmt.Printf("[%s] complexity=%s strategy=%q TopK=%d rw=%v hyb=%v rr=%v mq=%v\n",
		requestID, complexity, strategy.Description,
		strategy.TopK, strategy.UseRewriting, strategy.UseHybrid,
		strategy.UseRerank, strategy.UseMultiQuery)

	effectiveCfg := cfg
	effectiveCfg.Retrieval.TopK = strategy.TopK
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
	var multiQueryResults []map[string]interface{}
	rewriter := query.NewQueryRewriter(&cfg)

	fmt.Printf("Рерайтер создан, EnableRewriting=%v, EnableHyDE=%v\n",
		cfg.Retrieval.EnableRewriting, cfg.Retrieval.EnableHyDE)

	if cfg.Retrieval.EnableHyDE { // если включен HyDE то генерирую гипотетический ответ
		hypothetical, err := rewriter.GenerateHyDE(ctx, question)
		if err == nil && len(hypothetical) > 20 {
			queryForSearch = hypothetical
			fmt.Printf("HyDE сгенерирован (длина: %d символов)\n", len(hypothetical))
		} else if err != nil {
			fmt.Printf("HyDE ошибка: %v, используем оригинал\n", err)
		}
	}

	if cfg.Retrieval.EnableRewriting && queryForSearch == question { //если HyDE не использовался
		rewritten, err := rewriter.Rewrite(ctx, question, history)
		if err == nil && len(rewritten) > 5 && rewritten != question {
			queryForSearch = rewritten
			fmt.Printf("Запрос переписан: '%s' → '%s'\n", question, rewritten)
		} else if err != nil {
			fmt.Printf("Rewriting ошибка: %v\n", err)
		}
	}

	if queryForSearch != question { //финальный запрос
		fmt.Printf("Поиск по запросу: '%s'\n", queryForSearch)
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
		
		if cfg.Retrieval.EnableMultiQuery {   //muiti qery
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

		startEmbed := time.Now() //эмбединг
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

		startSearch := time.Now()  //поиск векторный
		results, err = vectorClient.Search(ctx, vector.CollectionName, vec32, cfg.Retrieval.TopK, userID)
		if err != nil || len(results) == 0 {
			return []string{}, []string{}, []float64{}, "ничего не нашла", []int{}, []string{}, 0, map[string]float64{}
		}
		searchDuration = time.Since(startSearch).Seconds()
		monitorMetrics.SetChunksFound(len(results))
		monitorMetrics.SetSearchDuration(time.Since(startSearch))

		if len(multiQueryResults) > 0 {
			results = multiQueryResults
			fmt.Printf("Использую Multi-Query результаты: %d документов\n", len(results))
		}

		textResults := []map[string]interface{}{}  // тестовый поиск HYBRID
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

		pipelineLog.Retrieval = &logger.RetrievalLog{  //логинг поиск
			VectorResults: len(results),
			TextResults:   len(textResults),
			FusedResults:  len(results),
			FromCache:     fromCache,
			DurationMs:    time.Since(startSearch).Milliseconds(),
		}
	
		beforeCount := len(results) // реранкинг
		rerankStart := time.Now()

		if cfg.Retrieval.EnableRerank && len(results) > cfg.Retrieval.TopK {
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
				reranker := rerank.NewReranker(&cfg)
				indices, _, err := reranker.Rerank(ctx, queryForSearch, documents, cfg.Retrieval.TopK)

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
				}
			}
		}

		pipelineLog.Rerank = &logger.RerankLog{  //логинг реранкинг
			BeforeCount: beforeCount,
			AfterCount:  len(results),
			DurationMs:  time.Since(rerankStart).Milliseconds(),
		}

		if cfg.Cache.EnableSearchCache && len(results) > 0 {  //сохр в кеш
			searchCache := cache.GetSearchCache()
			searchCache.Set(queryForSearch, userID, results)
			fmt.Printf("Результаты поиска сохранены в кэш\n")
		}
		
	}

	found := false //фильтрация по порогу
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
			return texts, docs, scores, "LLM не отвечает", pages, chunkIDs, 0, map[string]float64{}
		}
		fmt.Printf("LLM ответила, длина: %d символов\n", len(answer))
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
	prommetrics.RequestDuration.WithLabelValues("ask").Observe(time.Since(startTotal).Seconds())
prommetrics.RetrievedChunks.Observe(float64(len(texts)))
prommetrics.TokensUsed.Observe(float64(tokensUsed))

if len(texts) > 0 {
    prommetrics.RequestsTotal.WithLabelValues("ask", "success").Inc()
} else {
    prommetrics.RequestsTotal.WithLabelValues("ask", "empty").Inc()
}

	return texts, docs, scores, answer, pages, chunkIDs, tokensUsed, timings
}

func min(a, b int) int { // возвращает меньшее из двух чисел
	if a < b {
		return a
	}
	return b
}