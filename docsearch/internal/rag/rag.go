package rag

import (
    "fmt"
    "docsearch/internal/config"
    "docsearch/internal/corpus"
    "docsearch/internal/chunk"
    "docsearch/internal/embed"
    "docsearch/internal/retrieve"
    "docsearch/internal/llm"
)

func Index(cfg config.Config) {  //загрузка,режу,считаю эмбеддинг
    docs, err := corpus.LoadDocuments(cfg.Corpus.Path)
    if err != nil {
        fmt.Println("Ошибка загрузки:", err)
        return
    }

    fmt.Println("Количество документов:", len(docs))

    allText := []string{}
    allDocs := []string{}
    allVectors := [][]float64{}

    for i := 0; i < len(docs); i++ {
        doc := docs[i]
        parts := chunk.SplitText(doc.Text, cfg.Chunking.MaxTokens, cfg.Chunking.OverlapTokens, doc.Name)

        for j := 0; j < len(parts); j++ {
            one := parts[j]
            vec, err := embed.GetEmbedding(one.Text)   // получаю вектор через LM Studio
            if err != nil {
                continue
            }
            allText = append(allText, one.Text)
            allDocs = append(allDocs, doc.Name)
            allVectors = append(allVectors, vec)
    }
  }
}

func Ask(cfg config.Config, question string) ([]string, []string, []float64, string) {   // поиск по вопросу, нахожу похожие чанки и получаю ответ от LLM
    docs, err := corpus.LoadDocuments(cfg.Corpus.Path)
    if err != nil {
        return []string{}, []string{}, []float64{}, "Ошибка загрузки документов"
    }

    allText := []string{}
    allDocs := []string{}
    allVectors := [][]float64{}

    for i := 0; i < len(docs); i++ {
        doc := docs[i]
        parts := chunk.SplitText(doc.Text, cfg.Chunking.MaxTokens, cfg.Chunking.OverlapTokens, doc.Name)
        for j := 0; j < len(parts); j++ {
            one := parts[j]
            vec, err := embed.GetEmbedding(one.Text)
            if err != nil {
                continue
            }
            allText = append(allText, one.Text)
            allDocs = append(allDocs, doc.Name)
            allVectors = append(allVectors, vec)
        }
    }

    questionVec, err := embed.GetEmbedding(question)  // вектор для вопроса
    if err != nil {
        return []string{}, []string{}, []float64{}, "Ошибка получения вектора вопроса"
    }

    foundTexts, foundDocs, foundScores := retrieve.Search(allText,allDocs, allVectors, questionVec, cfg.Retrieval.TopK)    // ищу похожие чанки

    if len(foundTexts) == 0 {
        return foundTexts, foundDocs, foundScores, "Ответа в документации не найдено."
    }

   
    var answer string                        // проверяю, какой режим LLM включён
    if cfg.LLM.Provider == "mock" {
        answer = llm.GetMockAnswer(question, foundTexts)
    } else {
       
        answer, err = llm.GetAnswer(question, foundTexts)
        if err != nil {
            return foundTexts, foundDocs, foundScores, "Не удалось получить ответ от нейросети."
        }
    }

    return foundTexts, foundDocs, foundScores, answer
}
