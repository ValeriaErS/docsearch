package main

import (
    "fmt"
    "encoding/json"
    "os"
    "time"
    "docsearch/internal/config"
    "docsearch/internal/indexer"
    "docsearch/internal/rag"
    "docsearch/internal/vector"
    "context"
    "docsearch/internal/safety"
    "docsearch/internal/server"
    "docsearch/internal/eval"
)

type Source struct { // структура для json
    DocID string  `json:"doc_id"`
    Score float64 `json:"score"`
    Snippet string  `json:"snippet"`
    ChunkID string  `json:"chunk_id"`
}

type Response struct {
    Query string `json:"query"`
    Answer string `json:"answer"`
    Sources []Source `json:"sources"`
    Model string `json:"model"`
    TokensUsed int `json:"tokens_used"`
    DurationMs int64 `json:"duration_ms"`
}

func runDemo(cfg *config.Config, userID string, vectorClient vector.VectorStore) {
    fmt.Println("Демо режим")

    fmt.Println("Индексирую документы")
    idx := indexer.NewIndexer(cfg, vectorClient, userID)
    err := idx.Index(context.Background())
    if err != nil {
        fmt.Println("Ошибка индексации:", err)
        return
    }
    fmt.Println("Индексация завершена")
    if vc, ok := vectorClient.(*vector.FakeVectorStore); ok {
        fmt.Printf("После индексации: в FakeVectorStore %d точек\n", len(vc.Points))
    }

    questions := []string{
        "What is RAG?",
        "How to install DocSearch?",
        "How to install Linux?",
    }

    for _, q := range questions {
        fmt.Printf("\n Вопрос: %s\n", q)
if vc, ok := vectorClient.(*vector.FakeVectorStore); ok {
            fmt.Printf("Перед поиском: в FakeVectorStore %d точек, ищем для userID=%s\n", len(vc.Points), userID)
        }

         _, docs, scores, answer, _, _, tokensUsed, _ := rag.Ask(context.Background(), *cfg, q, userID, []map[string]string{}, vectorClient)
        found := false
        for i := 0; i < len(scores); i++ {
            if scores[i] >= cfg.Retrieval.MinScore {
                found = true
                break
            }
        }

        if !found || answer == "" {
            fmt.Println("Ответ: В документации нет информации по этому вопросу")
        } else {
            fmt.Printf("Ответ: %s\n", answer)
            fmt.Printf("Источников: %d\n", len(docs))
            fmt.Printf("Токенов: %d\n", tokensUsed)
        }
    }

    fmt.Println("\n Демо все")
}

func main() {
    args := os.Args[1:] //что ввел кроме 1
    configFile := "configs/config.yml"
    needIndex := false
    question := ""
    outFile := ""
    serveMode := false
    port := ":8080"
    userID := ""
    evalMode := false
    datasetPath := ""

    for i := 0; i < len(args); i++ { // разбираю команды
        if args[i] == "--config" && i+1 < len(args) {
            configFile = args[i+1]
            i = i + 1
        } else if args[i] == "index" {
            needIndex = true
        } else if args[i] == "ask" && i+1 < len(args) {
            question = args[i+1]
            i = i + 1
        } else if args[i] == "--out" && i+1 < len(args) {
            outFile = args[i+1]
            i = i + 1
        } else if args[i] == "web" {
            serveMode = true
        } else if args[i] == "--serve" {
            serveMode = true
        } else if args[i] == "--port" && i+1 < len(args) {
            port = args[i+1]
            i = i + 1
        } else if args[i] == "--user" && i+1 < len(args) {
            userID = args[i+1]
            i = i + 1
        } else if args[i] == "eval" {
            evalMode = true
        } else if args[i] == "--dataset" && i+1 < len(args) {
            datasetPath = args[i+1]
            i = i + 1
        } else if args[i] == "compare" {
            cfg, err := config.LoadConfig(configFile)
            if err != nil {
                fmt.Println("Ошибка загрузки конфига:", err)
                return
            }
            var vectorClient vector.VectorStore
            if cfg.Embeddings.Provider == "mock" {
                vectorClient = vector.NewFakeVectorStore()
            } else {
                realClient, err := vector.NewQdrantClient()
                if err != nil {
                    fmt.Println("Ошибка подключения к Qdrant:", err)
                    return
                }
                vectorClient = realClient
            }

            userID := "demo"
            for i, arg := range os.Args {
                if arg == "--user" && i+1 < len(os.Args) {
                    userID = os.Args[i+1]
                    break
                }
            }

            eval.CompareANNvsExact(cfg, userID, vectorClient)
            return
        } else if args[i] == "demo" {
            cfg, err := config.LoadConfig(configFile)
            if err != nil {
                fmt.Println("Ошибка загрузки конфига:", err)
                return
            }
            fakeClient := vector.NewFakeVectorStore()
            fmt.Println("Использую FakeVectorStore (mock-режим)")
            runDemo(cfg, "demo", fakeClient)
            return
        }
    }

    cfg, err := config.LoadConfig(configFile)
    if err != nil {
        fmt.Println("Ошибка загрузки конфига:", err)
        return
    }

    var sharedVectorClient vector.VectorStore //общий клиент
    if cfg.Embeddings.Provider == "mock" {
        sharedVectorClient = vector.NewFakeVectorStore()
        fmt.Println("Использую FakeVectorStore (mock-режим)")
    } else {
        realClient, err := vector.NewQdrantClient()
        if err != nil {
            fmt.Println("Ошибка подключения к Qdrant:", err)
            return
        }
        if err := realClient.Ping(context.Background()); err != nil {
            fmt.Println("Ошибка: Qdrant не отвечает:", err)
            return
        }
        sharedVectorClient = realClient
        if qdrantClient, ok := sharedVectorClient.(*vector.QdrantClient); ok {
            qdrantClient.VectorSize = cfg.Embeddings.VectorSize
        }
    }

    if evalMode {
        eval.RunEval(cfg, datasetPath, sharedVectorClient)
        return
    }

    if serveMode { // если запускаю сервер
        server.RunWeb(cfg, port, sharedVectorClient)
        return
    }

    if needIndex { // если нада индексировать
        if userID == "" { // проверка указали ли пользователя
            fmt.Println("Ошибка:для индексации нужно указать --user Имя")
            return
        }
        safeUser, err := safety.SanitizeAndValidateUser(userID)
        if err != nil {
            fmt.Println("Ошибка: неверное имя пользователя:", err)
            return
        }
        userID = safeUser

        fmt.Println("Передаю размер в индексер:", cfg.Embeddings.VectorSize)

        idx := indexer.NewIndexer(cfg, sharedVectorClient, userID)
        err = idx.Index(context.Background())
        if err != nil {
            fmt.Println("Ошибка индексации:", err)
            return
        }

        fmt.Println("С индексацией все хорошо")
        return
    }

    if question != "" { // если задан вопрос
        if userID == "" { // пользователь обяхателен для ask
            fmt.Println("Ошибка: для поиска необходимо указать пользователя")
            fmt.Println("Используйте: docsearch.exe ask \"вопрос\" --user Имя")
            fmt.Println("Пример: docsearch.exe ask \"Что такое RAG?\" --user Валерия")
            return
        }
        safeUser, err := safety.SanitizeAndValidateUser(userID)
        if err != nil {
            fmt.Println("Ошибка: неверное имя пользователя:", err)
            return
        }
        userID = safeUser

        startTime := time.Now()

       results, docs, scores, answer, _, chunkIDs, tokensUsed, _ := rag.Ask(context.Background(), *cfg, question, userID, []map[string]string{}, sharedVectorClient)
        found := false // проверяю порог
        for i := 0; i < len(scores); i++ {
            if scores[i] >= cfg.Retrieval.MinScore {
                found = true
                break
            }
        }

        if !found { // возврат JSON с пустым sources
            resp := Response{
                Query: question,
                Answer: "В документации нет информации по этому вопросу",
                Sources: []Source{},
                Model: cfg.LLM.Model,
                TokensUsed: 0,
                DurationMs: 0,
            }
            jsonData, _ := json.MarshalIndent(resp, "", "  ")
            fmt.Println(string(jsonData))
            return
        }

        var sources []Source // собираю источники
        for i := 0; i < len(results); i++ {
            snippet := results[i]
            if len(snippet) > 100 {
                snippet = snippet[:100] + "..."
            }
            sources = append(sources, Source{
                DocID: docs[i],
                Score: scores[i],
                Snippet: snippet,
                ChunkID: chunkIDs[i],
            })
        }

        duration := time.Since(startTime).Milliseconds()

        resp := Response{ // собираю ответ
            Query: question,
            Answer: answer,
            Sources: sources,
            Model: cfg.LLM.Model,
            TokensUsed: tokensUsed,
            DurationMs: duration,
        }

        jsonData, err := json.MarshalIndent(resp, "", "  ")
        if err != nil {
            fmt.Println("Ошибка формирования:", err)
            return
        }

        if outFile != "" {
            err := os.WriteFile(outFile, jsonData, 0644)
            if err != nil {
                fmt.Println("Ошибка сохранения в файл:", err)
            } else {
                fmt.Println("Результат сохранён в", outFile)
            }
        } else {
            fmt.Println(string(jsonData))
        }
        return
    }

    fmt.Println("Команды:")
    fmt.Println("index - индексация документов")
    fmt.Println("ask 'текст'- поиск по документации")
    fmt.Println("ask 'текст' --out file.json - сохранить результат в JSON")
    fmt.Println("serve - запустить HTTP сервер")
    fmt.Println("port :8080 - порт для сервера")
    fmt.Println("web - запустить веб-интерфейс")
    fmt.Println("eval - оценка качества поиска")
}