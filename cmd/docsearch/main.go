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
)

func main() {
    args := os.Args[1:]   //что ввел кроме 1 
    configFile := "configs/config.yml"
    needIndex := false
    question := ""
    outFile := ""
    serveMode := false
    port := ":8080"
    userID := ""
    evalMode := false

    for i := 0; i < len(args); i++ {   // разбираю команды
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
        } else if args[i]=="web"{
            serveMode=true
        } else if args[i] == "--serve" {
            serveMode = true
        } else if args[i] == "--port" && i+1 < len(args) {
            port = args[i+1]
            i = i + 1
        } else if args[i]=="--user" && i+1<len(args){
            userID=args[i+1]
            i = i + 1
        } else if args[i] == "eval" {
            evalMode = true 
        }
}

    cfg, err := config.LoadConfig(configFile)
    if err != nil {
        fmt.Println("Ошибка загрузки конфига:", err)
        return
    }
    if evalMode {
        runEval(cfg)
        return
    }

    if serveMode {  // если запускаю сервер
        runWeb(cfg, port)
        return
    }

    if needIndex {   // если нада индексировать
        fmt.Println("Передаю размер в индексер:", cfg.Embeddings.VectorSize) 
        vc := vector.NewQdrantClient()
        vc.VectorSize=cfg.Embeddings.VectorSize
        idx := indexer.NewIndexer(cfg, vc, userID)
        err = idx.Index()
        if err != nil {
            fmt.Println("Ошибка индексации:", err)
            return
        }

        fmt.Println("С индексацией все хорошо")
        return
    }
    if len(args) > 0 && args[0] == "eval" {
        runEval(cfg)
        return
    }


    if question != "" {    // если задан вопрос
        startTime := time.Now()

        results, docs, scores, answer := rag.Ask(*cfg, question, userID)

        found := false     // проверяю порог
        for i := 0; i < len(scores); i++ {
            if scores[i] >= cfg.Retrieval.MinScore {
                found = true
                break
            }
        }

        if !found {
            fmt.Println("Ответа в документации нет.")
            return
        }

        type Source struct {    // структура для json
            DocID string `json:"doc_id"`
            Score float64 `json:"score"`
            Snippet string `json:"snippet"`
        }

        type Response struct {
            Query string `json:"query"`
            Answer string `json:"answer"`
            Sources []Source `json:"sources"`
            Model string `json:"model"`
            TokensUsed int `json:"tokens_used"`
            DurationMs int64 `json:"duration_ms"`
        }

        var sources []Source           // собираю источники
        for i := 0; i < len(results); i++ {
            snippet := results[i]
            if len(snippet) > 100 {
                snippet = snippet[:100] + "..."
            }
            sources = append(sources, Source{
                DocID: docs[i],
                Score: scores[i],
                Snippet: snippet,
            })
        }

        duration := time.Since(startTime).Milliseconds()

        resp := Response{    // собираю ответ
            Query:question,
            Answer:answer,
            Sources:sources,
            Model:cfg.LLM.Model,
            TokensUsed:512,
            DurationMs:duration,
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