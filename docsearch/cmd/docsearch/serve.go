package main
import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "docsearch/internal/config"
    "docsearch/internal/rag"
)

func runServe(configFile string, port string) {
    cfg, err:= config.LoadConfig(configFile)           //загрузка настроек
    if err!= nil {
    fmt.Println("Ошибка загрузки настроек:", err)
    return
    }

    http.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {  //обработка запроса
        if r.Method!= "POST" {
        http.Error(w, "Нужно использовать POST", http.StatusMethodNotAllowed)
        return
        }

        var body struct {
        Query string `json:"query"`  //вопрос
        }
        err:= json.NewDecoder(r.Body).Decode(&body)   // читаю тело
        if err!= nil {
        http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
        return
        }

        if body.Query == "" {                                          //проверка что не пусто
        http.Error(w, "Нужно передать query", http.StatusBadRequest)
        return
        }

        startTime:= time.Now() 

        texts, docs, scores, answer:= rag.Ask(*cfg, body.Query)

        found:= false                                            //проверка порога
        for i:= 0; i < len(scores); i++ {
            if scores[i] >= cfg.Retrieval.MinScore {
                found = true
                break
            }
        }

        type Source struct {
            DocID string  `json:"doc_id"`
            Score float64 `json:"score"`
            Snippet string `json:"snippet"`
        }

        var sources []Source
        if found {
            for i:= 0; i < len(texts); i++ {                 //сборка источников
                snippet:= texts[i]
                if len(snippet) > 100 {
                    snippet = snippet[:100] + "..."
                }
                sources = append(sources, Source{
                    DocID:docs[i],
                    Score:scores[i],
                    Snippet:snippet,
                })
            }
        }

        duration:= time.Since(startTime).Milliseconds() 

        response:= struct {                             //ответ
            Query string `json:"query"`
            Answer string `json:"answer"`
            Sources []Source `json:"sources"`
            Model string `json:"model"`
            TokensUsed int `json:"tokens_used"`
            DurationMs int64 `json:"duration_ms"`
        }{
            Query:body.Query,
            Answer:answer,
            Sources:sources,
            Model:cfg.LLM.Model,   
            TokensUsed:512,             
            DurationMs:duration,        
        }

        w.Header().Set("Content-Type", "application/json")  //отправляю JSON
        json.NewEncoder(w).Encode(response)
    })

    fmt.Println("Сервер запущен на порту", port)

    err = http.ListenAndServe(port, nil)                                    // запускаю сервер
    if err != nil {
        fmt.Println("Ошибка запуска сервера:", err)
    }
}
