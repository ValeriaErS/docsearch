package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "docsearch/internal/config"
    "docsearch/internal/db"
    "docsearch/internal/embed"
    "docsearch/internal/llm"
    "docsearch/internal/vector"
)

var chatHistory = make(map[string][]map[string]string)
var database *db.DB

func runWeb(cfg *config.Config, port string) {
    var err error
    database, err = db.NewDB()
    if err != nil {
        fmt.Println("Ошибка базы:", err)
        return
    }
    defer database.Close()

    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {  // странички
        http.ServeFile(w, r, "web/index.html")
    })

    http.HandleFunc("/chat.html", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "web/chat.html")
    })

    http.HandleFunc("/test.html", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "web/test.html")
    })

    http.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "web/login.html")
    })

    http.HandleFunc("/register.html", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "web/register.html")
    })

    
    http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {   // логин
        if r.Method != "POST" {
            http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
            return
        }

        var req struct {
            Username string `json:"username"`
            Password string `json:"password"`
        }

        err := json.NewDecoder(r.Body).Decode(&req)
        if err != nil {
            http.Error(w, "Ошибка чтения", http.StatusBadRequest)
            return
        }

        ok := database.CheckUser(req.Username, req.Password)

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": ok,
            "user": req.Username,
        })
    })

    
    http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {  // регистрация
        if r.Method != "POST" {
            http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
            return
        }

        var req struct {
            Username string `json:"username"`
            Password string `json:"password"`
        }

        err := json.NewDecoder(r.Body).Decode(&req)
        if err != nil {
            http.Error(w, "Ошибка чтения", http.StatusBadRequest)
            return
        }

        
        err = database.AddUser(req.Username, req.Password)  // Добавляю пользователя в базу
        if err != nil {
            http.Error(w, "Пользователь уже существует", http.StatusConflict)
            return
        }
        fmt.Println("👤 Регистрируем:", req.Username)
        userDir:="docs/"+req.Username
        os.MkdirAll(userDir,0755)
        fmt.Println("папка создана:",userDir)


        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": true,
            "user": req.Username,
        })
    })

    
    http.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {  // вопрос
        if r.Method != "POST" {
            http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
            return
        }

        var req struct {
            Query string `json:"query"`
            User  string `json:"user"`
        }

        err := json.NewDecoder(r.Body).Decode(&req)
        if err != nil {
            http.Error(w, "Ошибка чтения", http.StatusBadRequest)
            return
        }

        if req.Query == "" {
            http.Error(w, "Пустой вопрос", http.StatusBadRequest)
            return
        }

        userID := req.User
        if userID == "" {
            userID = "default"
        }

        if chatHistory[userID] == nil {
            chatHistory[userID] = []map[string]string{}
        }

        chatHistory[userID] = append(chatHistory[userID], map[string]string{
            "role": "user",
            "content": req.Query,
        })

        answer, sources := findAnswer(cfg, req.Query, userID)

        chatHistory[userID] = append(chatHistory[userID], map[string]string{
            "role": "assistant",
            "content": answer,
        })

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "answer":  answer,
            "sources": sources,
        })
    })

    fmt.Println("Сайт запущен: http://localhost" + port)
    http.ListenAndServe("0.0.0.0"+port, nil)
}

func findAnswer(cfg *config.Config, question string, userID string) (string, []map[string]interface{}) {
    client := vector.NewQdrantClient()
    client.VectorSize = cfg.Embeddings.VectorSize

    vec, err := embed.GetEmbedding(question)
    if err != nil {
        return "Ошибка: не могу понять вопрос", nil
    }

    vec32 := []float32{}
    for _, v := range vec {
        vec32 = append(vec32, float32(v))
    }

    results, err := client.Search("documents", vec32, 10, userID)
    if err != nil || len(results) == 0 {
        return "Ничего не нашла", nil
    }

    context := []string{}
    sources := []map[string]interface{}{}

    for _, r := range results {
        payload := r["payload"].(map[string]interface{})
        text := payload["chunk_text"].(string)
        context = append(context, text)
        sources = append(sources, map[string]interface{}{
            "doc_id": payload["doc_id"],
            "score":  r["score"],
        })
    }

    seen := map[string]bool{}
    uniqueSources := []map[string]interface{}{}
    for _, s := range sources {
        docID := s["doc_id"].(string)
        if !seen[docID] {
            seen[docID] = true
            uniqueSources = append(uniqueSources, s)
        }
    }
    sources = uniqueSources

    answer, err := llm.GetAnswerWithHistory(question, context, chatHistory[userID])
    if err != nil {
        fmt.Println("Ошибка LLM:", err)
        return "Ошибка: нейросеть не отвечает", sources
    }

    return answer, sources
}